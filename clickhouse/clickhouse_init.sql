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

create table rooms_queue (id UUID, sfu String) engine = Kafka()
  SETTINGS 
    kafka_broker_list = 'kafka-bootstrap:9092',
    kafka_topic_list = 'rooms',
    kafka_group_name = 'clickhouse',
    kafka_format = 'JSONEachRow',
    kafka_num_consumers = 4;

create table sessions_queue (
  id UUID,
  start_time UInt64,
  duration UInt64,
  uid UUID
) engine = Kafka()
  SETTINGS 
    kafka_broker_list = 'kafka-bootstrap:9092',
    kafka_topic_list = 'sessions',
    kafka_group_name = 'clickhouse',
    kafka_format = 'JSONEachRow',
    kafka_num_consumers = 4;

create table participants_queue (
  room_id UUID,
  session_id UUID
) engine = Kafka()
  SETTINGS 
    kafka_broker_list = 'kafka-bootstrap:9092',
    kafka_topic_list = 'participants',
    kafka_group_name = 'clickhouse',
    kafka_format = 'JSONEachRow',
    kafka_num_consumers = 4;

create MATERIALIZED VIEW room_consumer to rooms
    as select id, sfu from rooms_queue;

create MATERIALIZED VIEW sessions_consumer to sessions
    as select id, start_time, duration, uid from sessions_queue;

create MATERIALIZED VIEW participants_consumer to participants
    as select room_id, session_id from participants_queue;
