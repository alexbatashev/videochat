create table participants ON CLUSTER '{cluster}' (room_id UUID, session_id UUID) engine = MergeTree()
  ORDER BY tuple() SETTINGS index_granularity = 8192;

create table rooms ON CLUSTER '{cluster}' (id UUID, sfu String) engine = MergeTree()
  ORDER BY tuple() SETTINGS index_granularity = 8192;

create table sessions ON CLUSTER '{cluster}' (
  id UUID,
  start_time UInt64,
  duration UInt64,
  uid UUID
) engine = MergeTree()
  ORDER BY tuple() SETTINGS index_granularity = 8192;

create table rooms_queue ON CLUSTER '{cluster}' (id UUID, sfu String) engine = Kafka()
  SETTINGS 
    kafka_broker_list = 'kafka-bootstrap:9092',
    kafka_topic_list = 'rooms',
    kafka_group_name = 'clickhouse',
    kafka_format = 'JSONEachRow',
    kafka_num_consumers = 4;

create table sessions_queue ON CLUSTER '{cluster}' (
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

create table participants_queue ON CLUSTER '{cluster}' (
  room_id UUID,
  session_id UUID
) engine = Kafka()
  SETTINGS 
    kafka_broker_list = 'kafka-bootstrap:9092',
    kafka_topic_list = 'participants',
    kafka_group_name = 'clickhouse',
    kafka_format = 'JSONEachRow',
    kafka_num_consumers = 4;

create MATERIALIZED VIEW room_consumer ON CLUSTER '{cluster}' to rooms
    as select id, sfu from rooms_queue;

create MATERIALIZED VIEW sessions_consumer ON CLUSTER '{cluster}' to sessions
    as select id, start_time, duration, uid from sessions_queue;

create MATERIALIZED VIEW participants_consumer ON CLUSTER '{cluster}' to participants
    as select room_id, session_id from participants_queue;
