#!/usr/bin/env tarantool

local cartridge = require('cartridge')
local os = require('os')
local log = require('log')

function addUser(id, name)
  local router = cartridge.service_get('vshard-router').get()
  local bucket_id = router:bucket_id_mpcrc32(id)
  local data, err = router:callrw(bucket_id, 
            'cache_storage.addUser', 
            { id, name })
  if err ~= nil then
    log.error(err)
  end
end

function startSession(id, startTime, userId)
  local router = cartridge.service_get('vshard-router').get()
  local bucket_id = router:bucket_id_mpcrc32(id)
  local data, err = router:callrw(bucket_id, 
            'cache_storage.startSession', 
            { id, startTime, 0, userId })
  if err ~= nil then
    log.error(err)
  end
end

function commitSession(id, duration)
  local router = cartridge.service_get('vshard-router').get()
  local bucket_id = router:bucket_id_mpcrc32(id)
  local data, err = router:callrw(bucket_id, 
            'cache_storage.commitSession', 
            { id, duration })
  if err ~= nil then
    log.error(err)
  end
end

function createRoom(id, sfuName)
  local router = cartridge.service_get('vshard-router').get()
  local bucket_id = router:bucket_id_mpcrc32(id)
  local data, err = router:callrw(bucket_id, 
            'cache_storage.createRoom', 
            { id, sfuName })
  if err ~= nil then
    log.error(err)
  end
end

function getRoom(id)
  local router = cartridge.service_get('vshard-router').get()
  local bucket_id = router:bucket_id_mpcrc32(id)
  local data, err = router:callro(bucket_id, 
            'cache_storage.getRoom', 
            { id })
  if err ~= nil then
    log.error(err)
    return nil
  end
  return data
end

function addRoomParticipant(roomId, sessionId)
  local router = cartridge.service_get('vshard-router').get()
  local bucket_id = router:bucket_id_mpcrc32(roomId)
  local data, err = router:callrw(bucket_id, 
            'cache_storage.addRoomParticipant', 
            { roomId, sessionId })
end

function removeRoomParticipant(roomId, sessionId)
  local router = cartridge.service_get('vshard-router').get()
  local bucket_id = router:bucket_id_mpcrc32(roomId)
  local data, err = router:callrw(bucket_id, 
            'cache_storage.removeRoomParticipant', 
            { roomId, sessionId })
end
local function init(_)
    return true
end

local function stop()
    --
end

local function validate_config(conf_new, conf_old)
    --

    return true
end

local function apply_config(conf, opts)
    if opts.is_master then
      box.schema.user.create('rest', { if_not_exists = true })
      box.schema.user.grant('rest', 'read,write,execute,create,drop','universe', nil, {if_not_exists = true})
      box.schema.user.grant('guest', 'read,write,execute,create,drop','universe', nil, {if_not_exists = true})
        --
    end

    --

    return true
end

return {
    role_name = 'router',
    init = init,
    stop = stop,
    validate_config = validate_config,
    apply_config = apply_config,
    dependencies = {'cartridge.roles.vshard-router'}
}
