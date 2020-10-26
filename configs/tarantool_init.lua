box.cfg
{
    pid_file = nil,
    background = false,
    log_level = 5
}
local function start()
  box.schema.space.create('users')
  box.space.users:format({
    { name = 'id', type = 'string' },
    { name = 'name', type = 'string' }
  })
  box.space.users:create_index('primary', {
    type = 'hash',
    parts = {'id'}
  })


  box.schema.space.create('rooms')
  box.space.rooms:format({
    { name = 'id', type = 'string' },
    { name = 'participants', type = 'array' }
  })
  box.space.rooms:create_index('primary', {
    type = 'hash',
    parts = {'id'}
  })
end

return {
  start = start;
}
