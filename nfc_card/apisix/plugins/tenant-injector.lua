local core = require("apisix.core")
local jwt = require("resty.jwt")
local ngx = ngx

local plugin_name = "tenant-injector"

local schema = {
    type = "object",
    properties = {
        header_name = {
            type = "string",
            default = "X-Tenant-ID"
        }
    }
}

local _M = {
    version = 0.1,
    priority = 2900,
    name = plugin_name,
    schema = schema
}

function _M.check_schema(conf)
    return core.schema.check(schema, conf)
end

function _M.rewrite(conf, ctx)
    -- 1. 尝试从JWT中提取租户ID
    local auth_header = core.request.header(ctx, "Authorization")
    if auth_header then
        local token = auth_header:match("Bearer%s+(.+)")
        if token then
            local jwt_obj = jwt:load_jwt(token)
            if jwt_obj and jwt_obj.payload and jwt_obj.payload.merchantId then
                core.request.set_header(ctx, conf.header_name, jwt_obj.payload.merchantId)
                core.log.info("tenant-injector: set tenant_id from JWT: ", jwt_obj.payload.merchantId)
                return
            end
        end
    end

    -- 2. 尝试从查询参数中提取
    local args = core.request.get_uri_args(ctx)
    if args and args.tenant_id then
        core.request.set_header(ctx, conf.header_name, args.tenant_id)
        core.log.info("tenant-injector: set tenant_id from query args: ", args.tenant_id)
        return
    end

    -- 3. 尝试从路径变量中提取
    local uri_captures = ngx.ctx.uri_captures
    if uri_captures and uri_captures.tenant_id then
        core.request.set_header(ctx, conf.header_name, uri_captures.tenant_id)
        core.log.info("tenant-injector: set tenant_id from uri captures: ", uri_captures.tenant_id)
        return
    end

    core.log.warn("tenant-injector: could not find tenant_id")
end

return _M 