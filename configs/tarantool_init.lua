box.cfg
{
    pid_file = nil,
    background = false,
    log_level = 6
}

local mqtt = require('mqtt')
connection = mqtt.new()
local ok, emsg = connection:connect({host = 'rabbitmq', port = 5672})
if not ok then
  error('connect ->', emsg)
end

function addUser(id, name)
  box.space.users:insert{id, name}
  -- TODO store users somewhere
end

function startSession(id, startTime, userId)
  box.space.sessions:insert{id, startTime, 0, userId}
end

function commitSession(id, duration)
  local t = box.space.sessions:update(id, {{'=', 'duration', duration}})
  -- TODO send session somewhere
  connection:publish('sessions_exchange', json:encode(t))
end

function createRoom(id)
  box.space.rooms:insert{id, {}, {}, 0, 'sfu'}
  local room = {}
  room['id'] = id
  room['sfu'] = 'sfu'
  connection:publish('rooms_exchange', json:encode(room))
end

function addRoomParticipant(roomId, sessionId)
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
  connection:publish('participants_exchange', json:encode(part))
end

function tablefind(tab,el)
  for index, value in pairs(tab) do
    if value == el then
    return index
    end
  end
end


function removeRoomParticipant(roomId, sessionId)
  box.begin()
  local record = box.space.rooms:get(roomId)
  local active_participants = record['active_participants']
  local key = tablefind(active_participants, sessionId)
  table.remove(active_participants, key)
  box.space.rooms:update(roomId, {{'=', 'active_participants', active_participants}, {'-', 'active_count', 1}})
end

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
