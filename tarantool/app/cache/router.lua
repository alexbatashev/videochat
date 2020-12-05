#!/usr/bin/env tarantool

local cartridge = require('cartridge')

function addUser(id, name)
  local router = cartridge.service_get('vshard-router').get()
  local bucket_id = router:bucket_id_mpcrc32(id)
  local data, err = router:callrw(bucket_id, 
            'cache_storage.addUser', 
            { id, name })
end

function startSession(id, startTime, userId)
  local router = cartridge.service_get('vshard-router').get()
  local bucket_id = router:bucket_id_mpcrc32(id)
  local data, err = router:callrw(bucket_id, 
            'cache_storage.startSession', 
            { id, startTime, 0, userId })
  -- box.space.sessions:insert{id, startTime, 0, userId}
end

function commitSession(id, duration)
  local router = cartridge.service_get('vshard-router').get()
  local bucket_id = router:bucket_id_mpcrc32(id)
  local data, err = router:callrw(bucket_id, 
            'cache_storage.commitSession', 
            { id, duration })
  -- local t = box.space.sessions:update(id, {{'=', 'duration', duration}})
  -- TODO send session somewhere
  -- connection:publish('sessions_exchange', json.encode(t))
end

function createRoom(id, sfuName)
  local router = cartridge.service_get('vshard-router').get()
  local bucket_id = router:bucket_id_mpcrc32(id)
  local data, err = router:callrw(bucket_id, 
            'cache_storage.createRoom', 
            { id, sfuName })
  -- box.space.rooms:insert{id, {}, {}, 0, sfuName}
  -- local room = {}
  -- room['id'] = id
  -- room['sfu'] = 'sfu'
  -- connection:publish('rooms_exchange', json.encode(room))
end

function getRoom(id)
  local router = cartridge.service_get('vshard-router').get()
  local bucket_id = router:bucket_id_mpcrc32(id)
  local data, err = router:callrw(bucket_id, 
            'cache_storage.getRoom', 
            { id })
  -- local room = box.space.rooms:get{id}
  -- return room
  return data
end

function addRoomParticipant(roomId, sessionId)
  local router = cartridge.service_get('vshard-router').get()
  local bucket_id = router:bucket_id_mpcrc32(roomId)
  local data, err = router:callrw(bucket_id, 
            'cache_storage.addRoomParticipant', 
            { roomId, sessionId })
  -- box.begin()
  -- local record = box.space.rooms:get({roomId})
  -- local participants = record['participants']
  -- local active_participants = record['active_participants']
  -- table.insert(participants, sessionId)
  -- table.insert(active_participants, sessionId)
  -- box.space.rooms:update({roomId}, {{'=', 'participants', participants}, {'=', 'active_participants', active_participants}, {'+', 'active_count', 1}})
  -- box.commit()
  -- local part = {}
  -- part['room_id'] = roomId
  -- part['session_id'] = sessionId
  -- connection:publish('participants_exchange', json.encode(part))
end

-- function tablefind(tab,el)
--   for index, value in pairs(tab) do
--     if value == el then
--     return index
--     end
--   end
-- end

function removeRoomParticipant(roomId, sessionId)
  local router = cartridge.service_get('vshard-router').get()
  local bucket_id = router:bucket_id_mpcrc32(roomId)
  local data, err = router:callrw(bucket_id, 
            'cache_storage.removeRoomParticipant', 
            { roomId, sessionId })
  -- box.begin()
  -- local record = box.space.rooms:get(roomId)
  -- local active_participants = record['active_participants']
  -- local key = tablefind(active_participants, sessionId)
  -- table.remove(active_participants, key)
  -- box.space.rooms:update(roomId, {{'=', 'active_participants', active_participants}, {'-', 'active_count', 1}})
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
      box.schema.user.grant('rest', 'read,write,execute,create,drop','universe', {if_not_exists = true})
      box.schema.user.grant('guest', 'read,write,execute,create,drop','universe', {if_not_exists = true})
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
