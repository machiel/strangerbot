CREATE TABLE `reports` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `reporter_id` bigint(20) unsigned NOT NULL,
  `created_at` datetime NOT NULL,
  `report` text NOT NULL,
  PRIMARY KEY (`id`),
  KEY `user_id` (`user_id`),
  KEY `reporter_id` (`reporter_id`),
  CONSTRAINT `reports_ibfk_1` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `reports_ibfk_2` FOREIGN KEY (`reporter_id`) REFERENCES `users` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `users` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `chat_id` int(11) NOT NULL,
  `last_activity` datetime NOT NULL,
  `match_chat_id` int(11) DEFAULT NULL,
  `available` tinyint(1) NOT NULL DEFAULT '1',
  `register_date` datetime DEFAULT NULL,
  `previous_match` int(11) DEFAULT NULL,
  `allow_pictures` tinyint(1) NOT NULL,
  `banned_until` datetime DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `chat_id` (`chat_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
