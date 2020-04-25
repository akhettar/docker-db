CREATE TABLE shop (article INT DEFAULT '0000' NOT NULL, dealer  CHAR(20)      DEFAULT ''     NOT NULL, price   DECIMAL(16,2) DEFAULT '0.00' NOT NULL,PRIMARY KEY(article, dealer));
INSERT INTO shop VALUES (1,'A',3.45),(1,'B',3.99),(2,'A',10.99);

