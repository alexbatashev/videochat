create table participants (room_id UUID, session_id UUID) engine = MergeTree()
  ORDER BY tuple() SETTINGS index_granularity = 8192;

create table rooms (id UUID, sfu String) engine = MergeTree()
  ORDER BY tuple() SETTINGS index_granularity = 8192;

create table sessions (
  id UUID,
  start_time UInt64,
  duration UInt64,
  uid UUID
) engine = MergeTree()
  ORDER BY tuple() SETTINGS index_granularity = 8192;

create table rooms_queue (id UUID, sfu String) engine = RabbitMQ SETTINGS rabbitmq_host_port = 'rabbitmq:5672',
  rabbitmq_exchange_name = 'rooms_exchange',
  rabbitmq_exchange_type = 'fanout',
  rabbitmq_format = 'JSONEachRow',
  rabbitmq_num_consumers = 5;

create table sessions_queue (
  id UUID,
  start_time UInt64,
  duration UInt64,
  uid UUID
) engine = RabbitMQ SETTINGS rabbitmq_host_port = 'rabbitmq:5672',
  rabbitmq_exchange_name = 'sessions_exchange',
  rabbitmq_exchange_type = 'fanout',
  rabbitmq_format = 'JSONEachRow',
  rabbitmq_num_consumers = 5;

create table participants_queue (
  room_id UUID,
  session_id UUID
) engine = RabbitMQ SETTINGS rabbitmq_host_port = 'rabbitmq:5672',
  rabbitmq_exchange_name = 'participants_exchange',
  rabbitmq_exchange_type = 'fanout',
  rabbitmq_format = 'JSONEachRow',
  rabbitmq_num_consumers = 5;

create MATERIALIZED VIEW room_consumer to rooms
    as select id, sfu from rooms_queue;

create MATERIALIZED VIEW sessions_consumer to sessions
    as select id, start_time, duration, uid from sessions_queue;

create MATERIALIZED VIEW participants_consumer to participants
    as select room_id, session_id from participants_queue;
