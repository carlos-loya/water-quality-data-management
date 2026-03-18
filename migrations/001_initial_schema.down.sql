-- 001_initial_schema.down.sql

DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS calibration_records;
DROP TABLE IF EXISTS instruments;
DROP TABLE IF EXISTS sample_results;
DROP TABLE IF EXISTS permit_limits;
DROP TABLE IF EXISTS monitoring_locations;
DROP TABLE IF EXISTS facilities;
DROP TABLE IF EXISTS analytical_methods;
DROP TABLE IF EXISTS parameters;
DROP TABLE IF EXISTS units_of_measure;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS organizations;

DROP EXTENSION IF EXISTS btree_gist;
DROP EXTENSION IF EXISTS pgcrypto;
