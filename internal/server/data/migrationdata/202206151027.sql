CREATE TABLE migrations (id VARCHAR(255) PRIMARY KEY);
INSERT INTO migrations VALUES('SCHEMA_INIT');
-- skip all these migrations
INSERT INTO migrations VALUES('202203231621');
INSERT INTO migrations VALUES('202203241643');
INSERT INTO migrations VALUES('202203301642');
INSERT INTO migrations VALUES('202203301652');
INSERT INTO migrations VALUES('202203301643');
INSERT INTO migrations VALUES('202203301644');
INSERT INTO migrations VALUES('202203301645');
INSERT INTO migrations VALUES('202203301646');
INSERT INTO migrations VALUES('202203301647');
INSERT INTO migrations VALUES('202203301648');
INSERT INTO migrations VALUES('202204061643');
INSERT INTO migrations VALUES('202204111503');
INSERT INTO migrations VALUES('202204181613');
INSERT INTO migrations VALUES('202204211705');
INSERT INTO migrations VALUES('202204281130');
INSERT INTO migrations VALUES('202204291613');
INSERT INTO migrations VALUES('202206081027');

CREATE TABLE `providers` (`id` integer,`created_at` datetime,`updated_at` datetime,`deleted_at` datetime,`name` text,`url` text,`client_id` text,`client_secret` text,`created_by` integer,PRIMARY KEY (`id`));
