package services

import (
	"bytes"
	"content-service/internal/config"
	"content-service/internal/domain/entities"
	"content-service/internal/messaging"
	"content-service/internal/storage"
	"fmt"
	"io/ioutil"
	"log"
	"mime/multipart"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// 分辨率定义
type Resolution struct {
	Name   string
	Width  int
	Height int
	Suffix string
}

// TranscodeService 视频转码服务
type TranscodeService struct {
	db             *sqlx.DB
	storageService *storage.StorageService
	config         *config.Config
	kafkaProducer  *messaging.KafkaProducer
	tempDir        string
	resolutions    []Resolution
}

// NewTranscodeService 创建新的转码服务
func NewTranscodeService(
	db *sqlx.DB,
	storageService *storage.StorageService,
	config *config.Config,
	kafkaProducer *messaging.KafkaProducer,
) (*TranscodeService, error) {
	// 创建临时目录
	tempDir := filepath.Join(os.TempDir(), "video_transcode")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("创建临时目录失败: %w", err)
	}

	// 定义支持的分辨率
	resolutions := []Resolution{
		{Name: "720p", Width: 1280, Height: 720, Suffix: "_720p"},
		{Name: "480p", Width: 854, Height: 480, Suffix: "_480p"},
		{Name: "360p", Width: 640, Height: 360, Suffix: "_360p"},
	}

	return &TranscodeService{
		db:             db,
		storageService: storageService,
		config:         config,
		kafkaProducer:  kafkaProducer,
		tempDir:        tempDir,
		resolutions:    resolutions,
	}, nil
}

// TranscodeVideo 转码视频
func (s *TranscodeService) TranscodeVideo(videoID, tenantID string) error {
	// 查询视频信息
	video, err := s.getVideo(videoID, tenantID)
	if err != nil {
		return fmt.Errorf("获取视频信息失败: %w", err)
	}

	// 更新视频转码状态为处理中
	if err := s.updateTranscodeStatus(video.ID.String(), entities.TranscodeStatusProcessing); err != nil {
		return fmt.Errorf("更新转码状态失败: %w", err)
	}

	// 发送视频处理事件
	if s.kafkaProducer != nil {
		payload := messaging.VideoProcessingPayload{
			ID:           video.ID.String(),
			TenantID:     video.TenantID.String(),
			Status:       string(entities.TranscodeStatusProcessing),
			ProcessingAt: time.Now().Format(time.RFC3339),
		}
		if err := s.kafkaProducer.SendVideoProcessing(payload); err != nil {
			log.Printf("发送视频处理事件失败: %v", err)
		}
	}

	// 在新的goroutine中处理转码
	go func() {
		err := s.processTranscode(video)
		if err != nil {
			log.Printf("视频转码失败: %v", err)
			// 更新状态为失败
			if updateErr := s.updateTranscodeStatus(video.ID.String(), entities.TranscodeStatusFailed); updateErr != nil {
				log.Printf("更新视频转码状态失败: %v", updateErr)
			}
		}
	}()

	return nil
}

// processTranscode 处理视频转码
func (s *TranscodeService) processTranscode(video entities.Video) error {
	// 下载视频到临时目录
	inputPath, err := s.downloadVideo(video.FileKey)
	if err != nil {
		return fmt.Errorf("下载视频失败: %w", err)
	}
	defer os.Remove(inputPath) // 清理临时文件

	// 获取视频信息
	duration, width, height, err := s.getVideoInfo(inputPath)
	if err != nil {
		return fmt.Errorf("获取视频信息失败: %w", err)
	}

	// 发送内容审核请求
	err = s.sendContentModerationRequest(video, inputPath)
	if err != nil {
		log.Printf("发送内容审核请求失败，但将继续处理: %v", err)
	}

	// 生成视频封面
	coverPath, err := s.generateCover(inputPath, video.ID.String())
	if err != nil {
		log.Printf("生成视频封面失败: %v", err)
		// 继续处理，不中断主流程
	} else {
		// 上传封面到存储服务
		coverKey := fmt.Sprintf("%s/%s_cover.jpg", video.TenantID.String(), video.ID.String())
		if err := s.uploadFile(coverPath, coverKey); err != nil {
			log.Printf("上传封面失败: %v", err)
		} else {
			// 更新视频封面信息
			if err := s.updateVideoCover(video.ID.String(), coverKey); err != nil {
				log.Printf("更新视频封面信息失败: %v", err)
			}

			// 对封面也进行内容审核
			err = s.sendCoverModerationRequest(video, coverPath, coverKey)
			if err != nil {
				log.Printf("发送封面审核请求失败: %v", err)
			}
		}
		// 清理临时封面文件
		os.Remove(coverPath)
	}

	// 转码并优化视频
	optimizedPath := filepath.Join(s.tempDir, fmt.Sprintf("%s_optimized.mp4", video.ID.String()))
	if err := s.optimizeForWeb(inputPath, optimizedPath); err != nil {
		log.Printf("优化视频失败，将使用原始视频继续处理: %v", err)
		// 使用原始视频继续处理
		optimizedPath = inputPath
	} else {
		defer os.Remove(optimizedPath) // 清理临时优化文件
	}

	// 转码为多种分辨率
	var transcodedFiles []struct {
		path     string
		fileKey  string
		width    int
		height   int
		fileSize int64
		suffix   string
	}

	// 确定最终使用的分辨率列表
	var finalResolutions []Resolution
	// 如果原始视频分辨率低于某个目标分辨率，则跳过该分辨率
	for _, res := range s.resolutions {
		if width >= res.Width || height >= res.Height {
			finalResolutions = append(finalResolutions, res)
		}
	}

	// 如果没有合适的分辨率，至少保留一个最高的分辨率
	if len(finalResolutions) == 0 && len(s.resolutions) > 0 {
		finalResolutions = append(finalResolutions, s.resolutions[0])
	}

	// 对每个分辨率进行转码
	for _, res := range finalResolutions {
		outputPath := filepath.Join(s.tempDir, fmt.Sprintf("%s%s.mp4", video.ID.String(), res.Suffix))

		// 计算目标分辨率
		targetWidth, targetHeight := s.calculateDimensions(width, height, res.Width, res.Height)

		// 执行转码
		if err := s.transcodeToMP4(optimizedPath, outputPath, targetWidth, targetHeight); err != nil {
			log.Printf("转码到%s分辨率失败: %v", res.Name, err)
			continue
		}

		// 获取转码后文件大小
		fileInfo, err := os.Stat(outputPath)
		if err != nil {
			log.Printf("获取转码文件信息失败: %v", err)
			os.Remove(outputPath)
			continue
		}

		// 添加到转码文件列表
		fileKey := fmt.Sprintf("%s/%s%s.mp4", video.TenantID.String(), video.ID.String(), res.Suffix)
		transcodedFiles = append(transcodedFiles, struct {
			path     string
			fileKey  string
			width    int
			height   int
			fileSize int64
			suffix   string
		}{
			path:     outputPath,
			fileKey:  fileKey,
			width:    targetWidth,
			height:   targetHeight,
			fileSize: fileInfo.Size(),
			suffix:   res.Suffix,
		})
	}

	// 上传转码后的文件
	var mainFileKey string
	for _, file := range transcodedFiles {
		if err := s.uploadFile(file.path, file.fileKey); err != nil {
			log.Printf("上传转码文件失败: %v", err)
			continue
		}

		// 第一个成功上传的文件作为主文件
		if mainFileKey == "" {
			mainFileKey = file.fileKey
		}

		// 清理临时文件
		os.Remove(file.path)
	}

	// 如果没有成功转码的文件，标记为失败
	if len(transcodedFiles) == 0 {
		return fmt.Errorf("没有成功转码的文件")
	}

	// 更新视频信息
	mainFile := transcodedFiles[0] // 使用第一个文件作为主信息
	if err := s.updateVideoInfo(video.ID.String(), duration, mainFile.width, mainFile.height, mainFile.fileSize, true, entities.TranscodeStatusCompleted); err != nil {
		return fmt.Errorf("更新视频信息失败: %w", err)
	}

	// 发送视频处理完成事件
	if s.kafkaProducer != nil {
		payload := messaging.VideoProcessedPayload{
			ID:          video.ID.String(),
			TenantID:    video.TenantID.String(),
			Status:      string(entities.TranscodeStatusCompleted),
			Duration:    duration,
			Width:       mainFile.width,
			Height:      mainFile.height,
			ProcessedAt: time.Now().Format(time.RFC3339),
		}
		if err := s.kafkaProducer.SendVideoProcessed(payload); err != nil {
			log.Printf("发送视频处理完成事件失败: %v", err)
		}
	}

	return nil
}

// downloadVideo 下载视频到临时目录
func (s *TranscodeService) downloadVideo(fileKey string) (string, error) {
	// 创建临时文件
	tempID := uuid.New().String()
	tempPath := filepath.Join(s.tempDir, fmt.Sprintf("input_%s%s", tempID, filepath.Ext(fileKey)))

	// 获取文件内容
	reader, err := s.storageService.GetObject(fileKey)
	if err != nil {
		return "", fmt.Errorf("获取视频文件失败: %w", err)
	}
	defer reader.Close()

	// 读取文件内容
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("读取视频内容失败: %w", err)
	}

	// 写入临时文件
	if err := ioutil.WriteFile(tempPath, data, 0644); err != nil {
		return "", fmt.Errorf("保存视频到临时文件失败: %w", err)
	}

	return tempPath, nil
}

// getVideoInfo 获取视频信息
func (s *TranscodeService) getVideoInfo(inputPath string) (float64, int, int, error) {
	// 使用ffprobe获取视频信息
	cmd := exec.Command(
		"ffprobe",
		"-v", "error",
		"-show_entries", "stream=width,height:format=duration",
		"-of", "csv=p=0",
		inputPath,
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return 0, 0, 0, fmt.Errorf("执行ffprobe失败: %v, %s", err, stderr.String())
	}

	// 解析输出
	output := stdout.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) < 3 {
		return 0, 0, 0, fmt.Errorf("ffprobe输出格式不正确: %s", output)
	}

	// 解析宽度、高度和时长
	width, err := strconv.Atoi(strings.TrimSpace(lines[0]))
	if err != nil {
		return 0, 0, 0, fmt.Errorf("解析视频宽度失败: %w", err)
	}

	height, err := strconv.Atoi(strings.TrimSpace(lines[1]))
	if err != nil {
		return 0, 0, 0, fmt.Errorf("解析视频高度失败: %w", err)
	}

	duration, err := strconv.ParseFloat(strings.TrimSpace(lines[2]), 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("解析视频时长失败: %w", err)
	}

	return duration, width, height, nil
}

// generateCover 生成视频封面
func (s *TranscodeService) generateCover(inputPath, videoID string) (string, error) {
	outputPath := filepath.Join(s.tempDir, fmt.Sprintf("%s_cover.jpg", videoID))

	// 使用ffmpeg在视频的1秒处截图作为封面
	cmd := exec.Command(
		"ffmpeg",
		"-i", inputPath,
		"-ss", "00:00:01.000",
		"-vframes", "1",
		"-q:v", "2",
		outputPath,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("生成封面失败: %v, %s", err, stderr.String())
	}

	return outputPath, nil
}

// transcodeToMP4 转码到MP4格式
func (s *TranscodeService) transcodeToMP4(inputPath, outputPath string, width, height int) error {
	// 增加压缩和优化参数
	// -crf 质量控制参数(0-51)，值越大压缩程度越高、质量越低，一般推荐18-28
	// -preset 压缩速度与质量的平衡，medium为平衡选项
	// -profile:v 编码配置文件，main是视频互联网传输的通用选项
	// -movflags 优化MP4容器结构
	// -pix_fmt 像素格式，yuv420p是最广泛支持的
	// -maxrate 限制最大码率
	// -bufsize 码率控制缓冲区大小

	cmd := exec.Command(
		"ffmpeg",
		"-i", inputPath,
		"-c:v", "libx264",
		"-crf", "23",
		"-preset", "medium",
		"-profile:v", "main",
		"-pix_fmt", "yuv420p",
		"-vf", fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2", width, height, width, height),
		"-c:a", "aac",
		"-b:a", "128k",
		"-movflags", "faststart",
		"-maxrate", "2M",
		"-bufsize", "4M",
		"-y",
		outputPath,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("转码失败: %v, %s", err, stderr.String())
	}

	return nil
}

// compressVideo 压缩视频文件以减小体积但保持质量
func (s *TranscodeService) compressVideo(inputPath, outputPath string) error {
	// 使用较高的CRF值减小文件大小，保持基本质量
	cmd := exec.Command(
		"ffmpeg",
		"-i", inputPath,
		"-c:v", "libx264",
		"-crf", "28",
		"-preset", "medium",
		"-c:a", "aac",
		"-b:a", "96k",
		"-movflags", "faststart",
		"-y",
		outputPath,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("压缩视频失败: %v, %s", err, stderr.String())
	}

	return nil
}

// optimizeForWeb 优化视频供网络传输
func (s *TranscodeService) optimizeForWeb(inputPath, outputPath string) error {
	// 使用适合网络传输的参数
	cmd := exec.Command(
		"ffmpeg",
		"-i", inputPath,
		"-c:v", "libx264",
		"-crf", "23",
		"-preset", "slower", // 更慢的编码速度但更好的压缩率
		"-profile:v", "main",
		"-level", "3.1", // 兼容大多数设备
		"-movflags", "+faststart", // 将元数据移到文件开头以便快速开始播放
		"-pix_fmt", "yuv420p",
		"-c:a", "aac",
		"-b:a", "128k",
		"-ac", "2", // 双声道
		"-r", "30", // 帧率30fps
		"-y",
		outputPath,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("优化视频失败: %v, %s", err, stderr.String())
	}

	return nil
}

// calculateDimensions 计算保持宽高比的目标尺寸
func (s *TranscodeService) calculateDimensions(srcWidth, srcHeight, maxWidth, maxHeight int) (int, int) {
	// 计算宽高比
	ratio := float64(srcWidth) / float64(srcHeight)

	// 根据目标尺寸计算新的宽度和高度
	var newWidth, newHeight int

	if float64(maxWidth)/float64(maxHeight) > ratio {
		// 高度限制
		newHeight = maxHeight
		newWidth = int(float64(newHeight) * ratio)
	} else {
		// 宽度限制
		newWidth = maxWidth
		newHeight = int(float64(newWidth) / ratio)
	}

	// 确保尺寸是偶数（ffmpeg要求）
	if newWidth%2 != 0 {
		newWidth--
	}
	if newHeight%2 != 0 {
		newHeight--
	}

	return newWidth, newHeight
}

// uploadFile 上传文件到存储服务
func (s *TranscodeService) uploadFile(filePath, fileKey string) error {
	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	// 获取文件信息
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("获取文件信息失败: %w", err)
	}

	// 创建临时的multipart.FileHeader
	tempHeader := &multipart.FileHeader{
		Filename: filepath.Base(filePath),
		Size:     fileInfo.Size(),
	}
	tempHeader.Header = make(map[string][]string)

	// 根据文件扩展名设置ContentType
	ext := strings.ToLower(filepath.Ext(filePath))
	contentType := "application/octet-stream"
	if ext == ".mp4" {
		contentType = "video/mp4"
	} else if ext == ".jpg" || ext == ".jpeg" {
		contentType = "image/jpeg"
	} else if ext == ".png" {
		contentType = "image/png"
	}
	tempHeader.Header.Set("Content-Type", contentType)

	// 关闭当前文件
	file.Close()

	// 上传文件
	return s.storageService.UploadFile(tempHeader, fileKey)
}

// getVideo 获取视频信息
func (s *TranscodeService) getVideo(videoID, tenantID string) (entities.Video, error) {
	var video entities.Video
	query := `
		SELECT * FROM videos 
		WHERE id = $1 AND tenant_id = $2
	`

	err := s.db.Get(&video, query, videoID, tenantID)
	if err != nil {
		return entities.Video{}, fmt.Errorf("查询视频失败: %w", err)
	}

	return video, nil
}

// updateTranscodeStatus 更新视频转码状态
func (s *TranscodeService) updateTranscodeStatus(videoID string, status entities.TranscodeStatus) error {
	query := `
		UPDATE videos 
		SET transcode_status = $1, updated_at = $2
		WHERE id = $3
	`

	_, err := s.db.Exec(query, status, time.Now(), videoID)
	if err != nil {
		return fmt.Errorf("更新转码状态失败: %w", err)
	}

	return nil
}

// updateVideoInfo 更新视频信息
func (s *TranscodeService) updateVideoInfo(videoID string, duration float64, width, height int, size int64, isTranscoded bool, status entities.TranscodeStatus) error {
	query := `
		UPDATE videos 
		SET duration = $1, width = $2, height = $3, size = $4, 
			is_transcoded = $5, transcode_status = $6, updated_at = $7
		WHERE id = $8
	`

	_, err := s.db.Exec(query, duration, width, height, size, isTranscoded, status, time.Now(), videoID)
	if err != nil {
		return fmt.Errorf("更新视频信息失败: %w", err)
	}

	return nil
}

// updateVideoCover 更新视频封面信息
func (s *TranscodeService) updateVideoCover(videoID, coverKey string) error {
	query := `
		UPDATE videos 
		SET cover_key = $1, updated_at = $2
		WHERE id = $3
	`

	_, err := s.db.Exec(query, coverKey, time.Now(), videoID)
	if err != nil {
		return fmt.Errorf("更新视频封面失败: %w", err)
	}

	return nil
}

// sendContentModerationRequest 发送视频内容审核请求
func (s *TranscodeService) sendContentModerationRequest(video entities.Video, videoPath string) error {
	if s.kafkaProducer == nil {
		return fmt.Errorf("Kafka生产者未初始化，无法发送内容审核请求")
	}

	// 创建审核载荷
	payload := messaging.SecurityCheckPayload{
		ID:          video.ID.String(),
		TenantID:    video.TenantID.String(),
		ContentType: "video",
		Result:      messaging.ContentModerationStatusPending,
		Violations:  make(map[string]string),
		CheckedAt:   time.Now().Format(time.RFC3339),
	}

	// 获取视频样本帧用于审核
	sampleFramesDir := filepath.Join(s.tempDir, fmt.Sprintf("%s_samples", video.ID.String()))
	err := os.MkdirAll(sampleFramesDir, 0755)
	if err != nil {
		return fmt.Errorf("创建样本帧目录失败: %w", err)
	}
	defer os.RemoveAll(sampleFramesDir)

	// 抽取视频帧进行内容审核
	err = s.extractFramesForModeration(videoPath, sampleFramesDir, 5) // 提取5个关键帧
	if err != nil {
		return fmt.Errorf("提取视频样本帧失败: %w", err)
	}

	// 发送内容审核事件
	if err := s.kafkaProducer.SendContentSecurityCheck(payload); err != nil {
		return fmt.Errorf("发送内容审核事件失败: %w", err)
	}

	return nil
}

// sendCoverModerationRequest 发送封面图片审核请求
func (s *TranscodeService) sendCoverModerationRequest(video entities.Video, coverPath, coverKey string) error {
	if s.kafkaProducer == nil {
		return fmt.Errorf("Kafka生产者未初始化，无法发送封面审核请求")
	}

	// 创建审核载荷
	payload := messaging.SecurityCheckPayload{
		ID:          video.ID.String() + "_cover",
		TenantID:    video.TenantID.String(),
		ContentType: "image",
		Result:      messaging.ContentModerationStatusPending,
		Violations:  make(map[string]string),
		CheckedAt:   time.Now().Format(time.RFC3339),
	}

	// 发送内容审核事件
	if err := s.kafkaProducer.SendContentSecurityCheck(payload); err != nil {
		return fmt.Errorf("发送封面审核事件失败: %w", err)
	}

	return nil
}

// extractFramesForModeration 从视频中提取帧用于内容审核
func (s *TranscodeService) extractFramesForModeration(videoPath, outputDir string, frameCount int) error {
	// 获取视频时长
	duration, _, _, err := s.getVideoInfo(videoPath)
	if err != nil {
		return fmt.Errorf("获取视频信息失败: %w", err)
	}

	// 计算均匀的时间点
	var timestamps []float64
	if duration <= 0 {
		return fmt.Errorf("视频时长无效")
	}

	interval := duration / float64(frameCount+1)
	for i := 1; i <= frameCount; i++ {
		timestamps = append(timestamps, interval*float64(i))
	}

	// 为每个时间点提取帧
	for i, timestamp := range timestamps {
		outputPath := filepath.Join(outputDir, fmt.Sprintf("frame_%d.jpg", i))

		// 使用ffmpeg在特定时间点提取帧
		cmd := exec.Command(
			"ffmpeg",
			"-ss", fmt.Sprintf("%.3f", timestamp),
			"-i", videoPath,
			"-vframes", "1",
			"-q:v", "2",
			"-y",
			outputPath,
		)

		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("在时间点%.3f提取帧失败: %v, %s", timestamp, err, stderr.String())
		}
	}

	return nil
}
