CREATE USER IF NOT EXISTS 'fleet'@'%' IDENTIFIED BY 'insecure'; 
CREATE DATABASE IF NOT EXISTS `mdm_apple`;
GRANT ALL PRIVILEGES ON mdm_apple.* TO 'fleet'@'%';
