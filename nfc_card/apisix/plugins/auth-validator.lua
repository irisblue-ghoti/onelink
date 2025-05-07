local core = require("apisix.core")
local jwt = require("resty.jwt")
local http = require("resty.http")
local cjson = require("cjson.safe")

local plugin_name = "auth-validator"

local schema = {
    type = "object",
    properties = {
        auth_service_url = {
            type = "string",
            default = "http://auth-service:3003/validate"
        },
        timeout = {
            type = "integer",
            minimum = 1000,
            default = 3000
        }
    }
}

local _M = {
    version = 0.1,
    priority = 2800,
    name = plugin_name,
    schema = schema
}

function _M.check_schema(conf)
    return core.schema.check(schema, conf)
end

function _M.rewrite(conf, ctx)
    -- 获取JWT令牌
    local auth_header = core.request.header(ctx, "Authorization")
    if not auth_header then
        return 401, { message = "Missing Authorization header" }
    end

    local token = auth_header:match("Bearer%s+(.+)")
    if not token then
        return 401, { message = "Invalid Authorization header format" }
    end

    -- 基本的JWT解析和验证
    local jwt_obj = jwt:verify(ngx.shared.jwt_keys, token)
    if not jwt_obj.verified then
        return 401, { message = "Invalid token: " .. jwt_obj.reason }
    end

    -- 与认证服务器进行验证
    local httpc = http.new()
    httpc:set_timeout(conf.timeout)
    
    local res, err = httpc:request_uri(conf.auth_service_url, {
        method = "POST",
        body = cjson.encode({ token = token }),
        headers = {
            ["Content-Type"] = "application/json"
        }
    })

    if not res then
        core.log.error("failed to validate token: ", err)
        return 500, { message = "Internal server error" }
    end

    if res.status ~= 200 then
        return res.status, { message = "Token validation failed" }
    end

    local body, err = cjson.decode(res.body)
    if not body then
        core.log.error("failed to decode response: ", err)
        return 500, { message = "Internal server error" }
    end

    -- 将验证服务返回的用户信息添加到请求上下文
    core.request.set_header(ctx, "X-User-ID", body.user_id)
    core.request.set_header(ctx, "X-User-Role", body.role)
    core.request.set_header(ctx, "X-Merchant-ID", body.merchant_id)

    -- 添加到ctx中供其他插件使用
    ctx.user_id = body.user_id
    ctx.user_role = body.role
    ctx.merchant_id = body.merchant_id
end

return _M 