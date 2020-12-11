local json = require('json')
local tnt_kafka = require('kafka')
local uuid = require('uuid')
local os = require('os')
local log = require('log')

local producer = nil

local function stop()
  --
  local err = producer:close() -- always stop consumer to send all pending messages before app close
    if err ~= nil then
        print(err)
        os.exit(1)
    end
end

local function validate_config(conf_new, conf_old)
  --

  return true
end

local function apply_config(conf, opts)
  if opts.is_master then
      --
  end

  --

  return true
end

local function tablefind(tab,el)
  for index, value in pairs(tab) do
    if value == el then
    return index
    end
  end
end

local cache_storage = {
addUser = function (id, name)
  box.space.users:insert{id, name}
  -- TODO store users somewhere
end,
startSession = function (id, startTime, userId)
  box.space.sessions:insert{id, startTime, 0, userId}
end,
commitSession = function (id, duration)
  local t = box.space.sessions:update(id, {{'=', 'duration', duration}})
  -- TODO send session somewhere
  print(json.encode(t))
  local message = json.encode(t)
  local err = producer:produce({ -- don't wait until message will be delivired to kafka
    topic = "sessions",
    key = uuid.str(),
    value = message
  })
  if err ~= nil then
    print(string.format("got error '%s' while sending value '%s'", err, message))
  else
    print(string.format("successfully sent value '%s'", message))
  end
end,
createRoom = function (id, sfuName)
  local room = {}
  room['id'] = id
  room['sfu'] = sfuName
  log.info(json.encode(room))
  local message = json.encode(room)
  local err = producer:produce({ -- don't wait until message will be delivired to kafka
    topic = "rooms",
    key = uuid.str(),
    -- value = id .. ',' .. sfuName -- hacky hack
    value = message
  })
  if err ~= nil then
    log.warn(string.format("got error '%s' while sending value '%s'", err, message))
    -- os.exit(1)
  else
    log.info(string.format("successfully sent value '%s'", message))
    -- os.exit(10)
  end
  box.space.rooms:insert{id, {}, {}, 0, sfuName}
end,
getRoom = function (id)
  local room = box.space.rooms:get{id}
  return room
end,
addRoomParticipant = function (roomId, sessionId)
  box.begin()
  local record = box.space.rooms:get({roomId})
  local participants = record['participants']
  local active_participants = record['active_participants']
  table.insert(participants, sessionId)
  table.insert(active_participants, sessionId)
  box.space.rooms:update({roomId}, {{'=', 'participants', participants}, {'=', 'active_participants', active_participants}, {'+', 'active_count', 1}})
  box.commit()
  local part = {}
  part['room_id'] = roomId
  part['session_id'] = sessionId
  log.info(json.encode(part))
  local message = json.encod(part)
  local err = producer:produce({ -- don't wait until message will be delivired to kafka
    topic = "participants",
    key = uuid.str(),
    -- value = '{ "room_id": "' .. roomId .. '", "session_id": "' .. sessionId .. '"}' -- hacky hack
    -- value = roomId .. ',' .. sessionId
    value = message
  })
  if err ~= nil then
    log.warn(string.format("got error '%s' while sending value '%s'", err, message))
  else
    log.info(string.format("successfully sent value '%s'", message))
  end
end,
removeRoomParticipant = function (roomId, sessionId)
  box.begin()
  local record = box.space.rooms:get(roomId)
  local active_participants = record['active_participants']
  local key = tablefind(active_participants, sessionId)
  table.remove(active_participants, key)
  box.space.rooms:update(roomId, {{'=', 'active_participants', active_participants}, {'-', 'active_count', 1}})
end,
}

local error_callback = function(err)
  log.error("got error: %s", err)
end
local log_callback = function(fac, str, level)
  log.info("got log: %d - %s - %s", level, fac, str)
end

local function init(opts)
  local err = nil
  producer, err = tnt_kafka.Producer.create({
    brokers = "kafka-bootstrap:9092", -- brokers for bootstrap
    options = {}, -- options for librdkafka
    error_callback = error_callback, -- optional callback for errors
    log_callback = log_callback -- optional callback for logs and debug messages
  })
  if err ~= nil then
    log.error(err)
    os.exit(1)
  end
  rawset(_G, 'cache_storage', cache_storage)
  if opts.is_master then
    box.schema.user.create('rest', { if_not_exists = true })
    box.schema.user.grant('rest', 'read,write,execute,create,drop','universe', nil, {if_not_exists = true})
    if not box.space.users then
      box.schema.space.create('users')
      box.space.users:format({
        { name = 'id', type = 'string' },
        { name = 'name', type = 'string' }
      })
      box.space.users:create_index('primary', {
        type = 'hash',
        parts = {'id'}
      })
    end

    if not box.space.sessions then
      box.schema.space.create('sessions')
      box.space.sessions:format({
        { name = 'id', type = 'string' },
        { name = 'start', type = 'unsigned' },
        { name = 'duration', type = 'unsigned' },
        { name = 'user_id', type = 'string' }
      })
      box.space.sessions:create_index('primary', {
        type = 'hash',
        parts = { 'id' }
      })
    end

    if not box.space.rooms then
      box.schema.space.create('rooms')
      box.space.rooms:format({
        { name = 'id', type = 'string' },
        { name = 'participants', type = 'array' },
        { name = 'active_participants', type = 'array' },
        { name = 'active_count', type = 'unsigned' },
        { name = 'sfu_worker', type = 'string' }
      })
      box.space.rooms:create_index('primary', {
        type = 'hash',
        parts = {'id'}
      })
    end

    for name, _ in pairs(cache_storage) do
        box.schema.func.create('cache_storage.' .. name, { setuid = true, if_not_exists = true })
        box.schema.user.grant('admin', 'execute', 'function', 'cache_storage.' .. name, { if_not_exists = true })
        box.schema.user.grant('rest', 'execute', 'function', 'cache_storage.' .. name, { if_not_exists = true })
        box.schema.user.grant('guest', 'execute', 'function', 'cache_storage.' .. name, { if_not_exists = true })
    end
  end
  return true
end

return {
  role_name = 'storage',
  init = init,
  stop = stop,
  validate_config = validate_config,
  apply_config = apply_config,
  dependencies = {'cartridge.roles.vshard-storage'}
}
