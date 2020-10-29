box.cfg
{
    pid_file = nil,
    background = false,
    log_level = 5
}

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

if not box.space.rooms then
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
