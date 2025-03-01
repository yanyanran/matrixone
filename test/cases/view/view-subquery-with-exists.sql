-- @suit

-- @case
-- @desc:test for  subquery with  exists
-- @label:bvt
-- @bvt:issue#3312
SELECT EXISTS(SELECT 1+1);
create view v1 as SELECT EXISTS(SELECT 1+1);
select * from v1;
drop view v1;

-- @bvt:issue
drop table if exists t1;
drop table if exists t2;
drop table if exists t3;
drop table if exists t4;
drop table if exists t5;
drop table if exists t6;
drop table if exists t7;
create table t1 (a int);
create table t2 (a int, b int);
create table t3 (a int);
create table t4 (a int not null, b int not null);
insert into t1 values (2);
insert into t2 values (1,7),(2,7);
insert into t4 values (4,8),(3,8),(5,9);
insert into t3 values (6),(7),(3);
select * from t3 where exists (select * from t2 where t2.b=t3.a);
select * from t3 where not exists (select * from t2 where t2.b=t3.a);
create view v1 as select * from t3 where exists (select * from t2 where t2.b=t3.a);
create view v2 as select * from t3 where not exists (select * from t2 where t2.b=t3.a);
select * from v1;
select * from v2;
drop view v1;
drop view v2;

insert into t4 values (12,7),(1,7),(10,9),(9,6),(7,6),(3,9),(1,10);
insert into t2 values (2,10);
create table t5 (a int);
insert into t5 values (5);
insert into t5 values (2);
create table t6 (patient_uq int, clinic_uq int);
create table t7( uq int primary key, name char(25));
insert into t7 values(1,"Oblastnaia bolnitsa"),(2,"Bolnitsa Krasnogo Kresta");
insert into t6 values (1,1),(1,2),(2,2),(1,3);
select * from t6 where exists (select * from t7 where uq = clinic_uq);
create view v1 as select * from t6 where exists (select * from t7 where uq = clinic_uq);
select * from v1;
drop view v1;


drop table if exists t1;
drop table if exists t2;
drop table if exists t3;
drop table if exists t4;
drop table if exists t5;
drop table if exists t6;
drop table if exists t7;
CREATE TABLE `t1` (
  `numeropost` int(8) unsigned NOT NULL,
  `maxnumrep` int(10) unsigned NOT NULL default 0,
  PRIMARY KEY  (`numeropost`)
);

INSERT INTO t1 (numeropost,maxnumrep) VALUES (40143,1),(43506,2);

CREATE TABLE `t2` (
      `mot` varchar(30) NOT NULL default '',
      `topic` int(8) unsigned NOT NULL default 0,
      `dt` date,
      `pseudo` varchar(35) NOT NULL default ''
    );

INSERT INTO t2 (mot,topic,dt,pseudo) VALUES ('joce','40143','2002-10-22','joce'), ('joce','43506','2002-10-22','joce');
SELECT numeropost,maxnumrep FROM t1 WHERE exists (SELECT 1 FROM t2 WHERE (mot='joce') AND dt >= date'2002-10-21' AND t1.numeropost = t2.topic) ORDER BY maxnumrep DESC LIMIT 0, 20;

create view v1 as SELECT numeropost,maxnumrep FROM t1 WHERE exists (SELECT 1 FROM t2 WHERE (mot='joce') AND dt >= date'2002-10-21' AND t1.numeropost = t2.topic) ORDER BY maxnumrep DESC LIMIT 0, 20;

select * from v1;
drop view v1;

drop table if exists t1;
drop table if exists t2;
drop table if exists t3;
CREATE TABLE `t1` (
  `mot` varchar(30) NOT NULL default '',
  `topic` int(8) unsigned NOT NULL default 0,
  `dt` date ,
  `pseudo` varchar(35) NOT NULL default ''
);
CREATE TABLE `t2` (
  `mot` varchar(30) NOT NULL default '',
  `topic` int(8) unsigned NOT NULL default 0,
  `dt` date,
  `pseudo` varchar(35) NOT NULL default ''
) ;
CREATE TABLE `t3` (
  `numeropost` int(8) unsigned NOT NULL,
  `maxnumrep` int(10) unsigned NOT NULL default 0,
  PRIMARY KEY  (`numeropost`)
);
INSERT INTO t1 VALUES ('joce','1',null,'joce'),('test','2',null,'test');
INSERT INTO t2 VALUES ('joce','1',null,'joce'),('test','2',null,'test');
INSERT INTO t3 VALUES (1,1);
SELECT DISTINCT topic FROM t2 WHERE NOT EXISTS(SELECT * FROM t3 WHERE numeropost=topic);
DELETE FROM t1 WHERE topic IN (SELECT DISTINCT topic FROM t2 WHERE NOT EXISTS(SELECT * FROM t3 WHERE numeropost=topic));
select * from t1;
create view v1 as select * from t1;
select * from v1;
drop view v1;
drop table if exists t1;
drop table if exists t2;
drop table if exists t3;

create table t1 (a int, b int);
insert into t1 values (1,2),(3,4);
select * from t1 up where exists (select * from t1 where t1.a=up.a);
create view v1 as select * from t1 up where exists (select * from t1 where t1.a=up.a);
select * from v1;
drop view v1;


drop table if exists t1;
drop table if exists t2;
drop table if exists t3;
CREATE TABLE t1 (a INT, b INT);
INSERT INTO t1 VALUES (1,1),(2,2);
CREATE TABLE t2 (a INT, b INT);
INSERT INTO t2 VALUES (1,1),(2,2);
CREATE TABLE t3 (a INT, b INT);
-- @bvt:issue#3307
SELECT COUNT(*) FROM t1 WHERE NOT EXISTS (SELECT 1 FROM t2 WHERE 1 = (SELECT MIN(t2.b) FROM t3)) ORDER BY COUNT(*);
SELECT COUNT(*) FROM t1 WHERE NOT EXISTS (SELECT 1 FROM t2 WHERE 1 = (SELECT MIN(t2.b) FROM t3)) ORDER BY COUNT(*);
create view v1 as SELECT COUNT(*) FROM t1 WHERE NOT EXISTS (SELECT 1 FROM t2 WHERE 1 = (SELECT MIN(t2.b) FROM t3)) ORDER BY COUNT(*);
create view v2 as SELECT COUNT(*) FROM t1 WHERE NOT EXISTS (SELECT 1 FROM t2 WHERE 1 = (SELECT MIN(t2.b) FROM t3)) ORDER BY COUNT(*);
select * from v1;
select * from v2;
drop view v1;
drop view v2;
-- @bvt:issue

drop table if exists t1;
drop table if exists t2;
drop table if exists t3;
CREATE TABLE t1 (f1 varchar(1));
INSERT INTO t1 VALUES ('v'),('s');
CREATE TABLE t2 (f1_key varchar(1));
INSERT INTO t2 VALUES ('j'),('v'),('c'),('m'),('d'),('d'),('y'),('t'),('d'),('s');
SELECT table1.f1, table2.f1_key FROM t1 AS table1, t2 AS table2
WHERE EXISTS
(
SELECT DISTINCT f1_key
FROM t2
WHERE f1_key != table2.f1_key AND f1_key >= table1.f1 );


create view v1 as SELECT table1.f1, table2.f1_key FROM t1 AS table1, t2 AS table2
WHERE EXISTS
(
SELECT DISTINCT f1_key
FROM t2
WHERE f1_key != table2.f1_key AND f1_key >= table1.f1 );
select * from v1;
drop view v1;

drop table if exists t1;
drop table if exists t2;
drop table if exists t3;
CREATE TABLE t1( pk int PRIMARY KEY,uk int,ukn int NOT NULL,ik int,d int);
INSERT INTO t1 VALUES (0, NULL, 0, NULL, NULL),(1, 10, 20, 30, 40),(2, 20, 40, 60, 80);
CREATE TABLE t2(pk int PRIMARY KEY);
INSERT INTO t2 VALUES (1), (2), (3), (4), (5), (6), (7), (8), (9),(10),
(11),(12),(13),(14),(15),(16),(17),(18),(19),(20),
(21),(22),(23),(24),(25),(26),(27),(28),(29),(30),
(31),(32),(33),(34),(35),(36),(37),(38),(39),(40),
(41),(42),(43),(44),(45),(46),(47),(48),(49),(50),
(51),(52),(53),(54),(55),(56),(57),(58),(59),(60),
(61),(62),(63),(64),(65),(66),(67),(68),(69),(70),
(71),(72),(73),(74),(75),(76),(77),(78),(79),(80);
SELECT 1  WHERE EXISTS (SELECT * FROM t1 AS it);
SELECT * FROM t1 AS ot WHERE EXISTS (SELECT 1);
SELECT * FROM t1 AS ot WHERE EXISTS (SELECT 1 WHERE FALSE);
SELECT * FROM t1 AS ot WHERE EXISTS (SELECT * FROM t1 AS it);
SELECT * FROM t1 AS ot WHERE EXISTS (SELECT * FROM t1 AS it WHERE it.pk = 1);
SELECT * FROM t1 AS ot WHERE EXISTS (SELECT * FROM t1 AS it WHERE it.uk = 1);
SELECT * FROM t1 AS ot WHERE EXISTS (SELECT * FROM t1 AS it WHERE it.ik = 1);
SELECT * FROM t1 AS ot WHERE EXISTS (SELECT * FROM t1 AS it WHERE it.d = 1);
SELECT * FROM t1 AS ot WHERE EXISTS (SELECT * FROM t1 AS it WHERE it.pk = ot.pk);
SELECT * FROM t1 AS ot WHERE EXISTS (SELECT * FROM t1 AS it WHERE it.uk = ot.uk);
SELECT * FROM t1 AS ot WHERE EXISTS (SELECT * FROM t1 AS it WHERE it.ukn = ot.ukn);
SELECT * FROM t1 AS ot WHERE EXISTS (SELECT * FROM t1 AS it WHERE it.d = ot.d);
SELECT * FROM t1 AS ot WHERE EXISTS (SELECT * FROM t1 AS it WHERE it.pk > ot.pk);
SELECT * FROM t1 AS ot WHERE EXISTS (SELECT * FROM t1 AS it WHERE it.uk > ot.uk);
SELECT * FROM t1 AS ot WHERE EXISTS (SELECT * FROM t1 AS it WHERE it.ukn > ot.ukn);
SELECT * FROM t1 AS ot WHERE EXISTS (SELECT * FROM t1 AS it WHERE it.ik > ot.ik);
SELECT * FROM t1 AS ot WHERE EXISTS (SELECT * FROM t1 AS it WHERE it.d > ot.d);
SELECT * FROM t2 AS ot WHERE EXISTS (SELECT * FROM t1 AS it WHERE it.pk = ot.pk);
SELECT * FROM t2 AS ot WHERE EXISTS (SELECT * FROM t1 AS it WHERE it.uk = ot.pk);
SELECT * FROM t2 AS ot WHERE EXISTS (SELECT * FROM t1 AS it WHERE it.ukn = ot.pk);
SELECT * FROM t2 AS ot WHERE EXISTS (SELECT * FROM t1 AS it WHERE it.d = ot.pk);
SELECT * FROM t1 AS ot WHERE EXISTS (SELECT * FROM t2 AS it WHERE ot.d = it.pk - 1);
SELECT * FROM t1 AS ot WHERE EXISTS (SELECT * FROM t1 AS it1 JOIN t2 AS it2 ON it1.pk > it2.pk WHERE ot.d = it2.pk);


create view v1 as SELECT 1  WHERE EXISTS (SELECT * FROM t1 AS it);
create view v2 as SELECT * FROM t1 AS ot WHERE EXISTS (SELECT 1);
create view v3 as SELECT * FROM t1 AS ot WHERE EXISTS (SELECT 1 WHERE FALSE);
create view v4 as SELECT * FROM t1 AS ot WHERE EXISTS (SELECT * FROM t1 AS it);
create view v5 as SELECT * FROM t1 AS ot WHERE EXISTS (SELECT * FROM t1 AS it WHERE it.pk = 1);
create view v6 as SELECT * FROM t1 AS ot WHERE EXISTS (SELECT * FROM t1 AS it WHERE it.uk = 1);
create view v7 as SELECT * FROM t1 AS ot WHERE EXISTS (SELECT * FROM t1 AS it WHERE it.ik = 1);
create view v8 as SELECT * FROM t1 AS ot WHERE EXISTS (SELECT * FROM t1 AS it WHERE it.d = 1);
create view v9 as SELECT * FROM t1 AS ot WHERE EXISTS (SELECT * FROM t1 AS it WHERE it.pk = ot.pk);
create view v10 as SELECT * FROM t1 AS ot WHERE EXISTS (SELECT * FROM t1 AS it WHERE it.uk = ot.uk);
create view v11 as SELECT * FROM t1 AS ot WHERE EXISTS (SELECT * FROM t1 AS it WHERE it.ukn = ot.ukn);
create view v12 as SELECT * FROM t1 AS ot WHERE EXISTS (SELECT * FROM t1 AS it WHERE it.d = ot.d);
create view v13 as SELECT * FROM t1 AS ot WHERE EXISTS (SELECT * FROM t1 AS it WHERE it.pk > ot.pk);
create view v14 as SELECT * FROM t1 AS ot WHERE EXISTS (SELECT * FROM t1 AS it WHERE it.uk > ot.uk);
create view v15 as SELECT * FROM t1 AS ot WHERE EXISTS (SELECT * FROM t1 AS it WHERE it.ukn > ot.ukn);
create view v16 as SELECT * FROM t1 AS ot WHERE EXISTS (SELECT * FROM t1 AS it WHERE it.ik > ot.ik);
create view v17 as SELECT * FROM t1 AS ot WHERE EXISTS (SELECT * FROM t1 AS it WHERE it.d > ot.d);
create view v18 as SELECT * FROM t2 AS ot WHERE EXISTS (SELECT * FROM t1 AS it WHERE it.pk = ot.pk);
create view v19 as SELECT * FROM t2 AS ot WHERE EXISTS (SELECT * FROM t1 AS it WHERE it.uk = ot.pk);
create view v20 as SELECT * FROM t2 AS ot WHERE EXISTS (SELECT * FROM t1 AS it WHERE it.ukn = ot.pk);
create view v21 as SELECT * FROM t2 AS ot WHERE EXISTS (SELECT * FROM t1 AS it WHERE it.d = ot.pk);
create view v22 as SELECT * FROM t1 AS ot WHERE EXISTS (SELECT * FROM t2 AS it WHERE ot.d = it.pk - 1);
create view v23 as SELECT * FROM t1 AS ot WHERE EXISTS (SELECT * FROM t1 AS it1 JOIN t2 AS it2 ON it1.pk > it2.pk WHERE ot.d = it2.pk);

select * from v1;
select * from v2;
select * from v3;
select * from v4;
select * from v5;
select * from v6;
select * from v7;
select * from v8;
select * from v9;
select * from v10;
select * from v11;
select * from v12;
select * from v13;
select * from v14;
select * from v15;
select * from v16;
select * from v17;
select * from v18;
select * from v19;
select * from v20;
select * from v21;
select * from v22;
select * from v23;

drop view v1;
drop view v2;
drop view v3;
drop view v4;
drop view v5;
drop view v6;
drop view v7;
drop view v8;
drop view v9;
drop view v10;
drop view v11;
drop view v12;
drop view v13;
drop view v14;
drop view v15;
drop view v16;
drop view v17;
drop view v18;
drop view v19;
drop view v20;
drop view v21;
drop view v22;
drop view v23;


drop table if exists t1;
drop table if exists t2;
drop table if exists t3;
CREATE TABLE t1 (a int);
SELECT * FROM t1 WHERE EXISTS (SELECT * FROM t1 WHERE 127 = 55);

-- @case
-- @desc:test for  subquery with group by and having
-- @label:bvt
drop table if exists t1;
drop table if exists t2;
drop table if exists t3;
create table t1 (s1 int);
create table t2 (s1 int);
insert into t1 values (1);
insert into t2 values (1);
select * from t1 where exists (select s1 from t2 group by s1 having max(t2.s1)=t1.s1);

create view v1 as select * from t1 where exists (select s1 from t2 group by s1 having max(t2.s1)=t1.s1);
select * from v1;
drop view v1;

drop table if exists t1;
drop table if exists t2;
create table t1 (id int not null, text varchar(20) not null default '', primary key (id));
insert into t1 (id, text) values (1, 'text1'), (2, 'text2'), (3, 'text3'), (4, 'text4'), (5, 'text5'), (6, 'text6'), (7, 'text7'), (8, 'text8'), (9, 'text9'), (10, 'text10'), (11, 'text11'), (12, 'text12');
select * from t1 as tt where not exists (select id from t1 where id < 8 and (id = tt.id or id is null) having id is not null);

create view v1 as select * from t1 as tt where not exists (select id from t1 where id < 8 and (id = tt.id or id is null) having id is not null);
select * from v1;
drop view v1;


drop table if exists t1;
drop table if exists t2;
drop table if exists t3;
CREATE TABLE t1 (id int NOT NULL, st CHAR(2));
INSERT INTO t1 VALUES (3,'FL'), (2,'GA'), (4,'FL'), (1,'GA'), (5,'NY'), (7,'FL'), (6,'NY');
CREATE TABLE t2 (id int NOT NULL);
INSERT INTO t2 VALUES (7), (5), (1), (3);
SELECT id, st FROM t1  WHERE st IN ('GA','FL') AND EXISTS(SELECT 1 FROM t2 WHERE t2.id=t1.id);
SELECT id, st FROM t1  WHERE st IN ('GA','FL') AND EXISTS(SELECT 1 FROM t2 WHERE t2.id=t1.id) GROUP BY id;
SELECT id, st FROM t1 WHERE st IN ('GA','FL') AND NOT EXISTS(SELECT 1 FROM t2 WHERE t2.id=t1.id);
SELECT id, st FROM t1 WHERE st IN ('GA','FL') AND NOT EXISTS(SELECT 1 FROM t2 WHERE t2.id=t1.id) GROUP BY id;

create view v1 as SELECT id, st FROM t1  WHERE st IN ('GA','FL') AND EXISTS(SELECT 1 FROM t2 WHERE t2.id=t1.id);
create view v2 as SELECT id, st FROM t1  WHERE st IN ('GA','FL') AND EXISTS(SELECT 1 FROM t2 WHERE t2.id=t1.id) GROUP BY id;
create view v3 as SELECT id, st FROM t1 WHERE st IN ('GA','FL') AND NOT EXISTS(SELECT 1 FROM t2 WHERE t2.id=t1.id);
create view v4 as SELECT id, st FROM t1 WHERE st IN ('GA','FL') AND NOT EXISTS(SELECT 1 FROM t2 WHERE t2.id=t1.id) GROUP BY id;

select * from v1;
select * from v2;
select * from v3;
select * from v4;

drop view v1;
drop view v2;
drop view v3;
drop view v4;

drop table if exists t1;
drop table if exists t2;
drop table if exists t3;
CREATE TABLE t1 (a INT, b INT);
INSERT INTO t1 VALUES (1, 2), (1,3), (1,4), (2,1), (2,2);
SELECT a1.a, COUNT(*) FROM t1 a1 WHERE a1.a = 1 AND EXISTS( SELECT a2.a FROM t1 a2 WHERE a2.a = a1.a) GROUP BY a1.a;
create view v1 as SELECT a1.a, COUNT(*) FROM t1 a1 WHERE a1.a = 1 AND EXISTS( SELECT a2.a FROM t1 a2 WHERE a2.a = a1.a) GROUP BY a1.a;
select * from v1;
drop view v1;
DROP TABLE if exists t1;


-- @case
-- @desc:test for  subquery with func
-- @label:bvt
drop table if exists t1;
drop table if exists t2;
CREATE TABLE t1 ( a int, b int );
INSERT INTO t1 VALUES (1,1),(2,2),(3,3);
-- @bvt:issue#3312
SELECT EXISTS(SELECT a FROM t1 WHERE b = 2 and a.a > t1.a) IS NULL from t1 a;
SELECT EXISTS(SELECT a FROM t1 WHERE b = 2 and a.a < t1.a) IS NOT NULL from t1 a;
SELECT EXISTS(SELECT a FROM t1 WHERE b = 2 and a.a = t1.a) IS NULL from t1 a;
create view v1 as SELECT EXISTS(SELECT a FROM t1 WHERE b = 2 and a.a > t1.a) IS NULL from t1 a;
create view v2 as SELECT EXISTS(SELECT a FROM t1 WHERE b = 2 and a.a < t1.a) IS NOT NULL from t1 a;
create view v3 as SELECT EXISTS(SELECT a FROM t1 WHERE b = 2 and a.a = t1.a) IS NULL from t1 a;
select * from v1;
select * from v2;
select * from v3;

drop view v1;
drop view v2;
drop view v3;
-- @bvt:issue
drop table if exists t1;

-- @case
-- @desc:test for  subquery with Arithmetic calculation
-- @label:bvt
drop table if exists t1;
create table t1 (df decimal(5,1));
insert into t1 values(1.1);
-- @bvt:issue#3312
select 1.1 * exists(select * from t1);
create view v1 as select 1.1 * exists(select * from t1);
select * from v1;
drop view v1;
-- @bvt:issue
drop table if exists t1;

-- @case
-- @desc:test for  subquery with uion
-- @label:bvt
drop table if exists t1;
CREATE TABLE t1 (i INT);
SELECT * FROM t1 WHERE NOT EXISTS
  (
   (SELECT i FROM t1) UNION
   (SELECT i FROM t1)
  );
SELECT * FROM t1 WHERE NOT EXISTS (((SELECT i FROM t1) UNION (SELECT i FROM t1)));

create view v1 as SELECT * FROM t1 WHERE NOT EXISTS
  (
   (SELECT i FROM t1) UNION
   (SELECT i FROM t1)
  );
create view v2 as SELECT * FROM t1 WHERE NOT EXISTS (((SELECT i FROM t1) UNION (SELECT i FROM t1)));
select * from v1;
select * from v2;

drop view v1;
drop view v2;

drop table if exists t1;
drop table if exists t2;
drop table if exists t3;
CREATE TABLE t1 (id char(4) PRIMARY KEY, c int);
CREATE TABLE t2 (c int);
INSERT INTO t1 VALUES ('aa', 1);
INSERT INTO t2 VALUES (1);
-- @bvt:issue#4354
SELECT * FROM t1
  WHERE EXISTS (SELECT c FROM t2 WHERE c=1
                UNION
                SELECT c from t2 WHERE c=t1.c);


create view v1 as SELECT * FROM t1
  WHERE EXISTS (SELECT c FROM t2 WHERE c=1
                UNION
                SELECT c from t2 WHERE c=t1.c);
select * from v1;
drop view v1;

-- @bvt:issue
INSERT INTO t1 VALUES ('bb', 2), ('cc', 3), ('dd',1);
-- @bvt:issue#4354
SELECT * FROM t1
  WHERE EXISTS (SELECT c FROM t2 WHERE c=1
                UNION
                SELECT c from t2 WHERE c=t1.c);
create view v1 as SELECT * FROM t1
  WHERE EXISTS (SELECT c FROM t2 WHERE c=1
                UNION
                SELECT c from t2 WHERE c=t1.c);
select * from v1;
drop view v1;
-- @bvt:issue
INSERT INTO t2 VALUES (2);
CREATE TABLE t3 (c int);
INSERT INTO t3 VALUES (1);
-- @bvt:issue#4354
SELECT * FROM t1
  WHERE EXISTS (SELECT t2.c FROM t2 JOIN t3 ON t2.c=t3.c WHERE t2.c=1
                UNION
                SELECT c from t2 WHERE c=t1.c);
create view v1 as SELECT * FROM t1
  WHERE EXISTS (SELECT t2.c FROM t2 JOIN t3 ON t2.c=t3.c WHERE t2.c=1
                UNION
                SELECT c from t2 WHERE c=t1.c);
select * from v1;
drop view v1;
-- @bvt:issue

drop table if exists t1;
drop table if exists t2;
drop table if exists t3;
CREATE TABLE t1 (a INT);
CREATE TABLE t2 (a INT);
INSERT INTO t1 VALUES (1),(2);
INSERT INTO t2 VALUES (1),(2);
SELECT 2 FROM t1 WHERE EXISTS ((SELECT 1 FROM t2 WHERE t1.a=t2.a));
SELECT 2 FROM t1 WHERE EXISTS ((SELECT 1 FROM t2 WHERE t1.a=t2.a) UNION (SELECT 1 FROM t2 WHERE t1.a = t2.a));

create view v1 as SELECT 2 FROM t1 WHERE EXISTS ((SELECT 1 FROM t2 WHERE t1.a=t2.a));
create view v2 as SELECT 2 FROM t1 WHERE EXISTS ((SELECT 1 FROM t2 WHERE t1.a=t2.a) UNION (SELECT 1 FROM t2 WHERE t1.a = t2.a));
select * from v1;
select * from v2;

drop view v1;
drop view v2;

drop table if exists t1;
drop table if exists t2;

-- @case
-- @desc:test for  subquery with join
-- @label:bvt
drop table if exists t1;
drop table if exists t2;
drop table if exists t3;
CREATE TABLE t1 ( c1 int );
INSERT INTO t1 VALUES ( 1 );
INSERT INTO t1 VALUES ( 2 );
INSERT INTO t1 VALUES ( 3 );
INSERT INTO t1 VALUES ( 6 );

CREATE TABLE t2 ( c2 int );
INSERT INTO t2 VALUES ( 1 );
INSERT INTO t2 VALUES ( 4 );
INSERT INTO t2 VALUES ( 5 );
INSERT INTO t2 VALUES ( 6 );

CREATE TABLE t3 ( c3 int );
INSERT INTO t3 VALUES ( 7 );
INSERT INTO t3 VALUES ( 8 );

SELECT c1,c2 FROM t1 LEFT JOIN t2 ON c1 = c2
  WHERE EXISTS (SELECT c3 FROM t3 WHERE c2 IS NULL );
  
create view v1 as SELECT c1,c2 FROM t1 LEFT JOIN t2 ON c1 = c2
  WHERE EXISTS (SELECT c3 FROM t3 WHERE c2 IS NULL );
select * from v1;
drop view v1;

drop table if exists t1;
drop table if exists t2;
drop table if exists t3;

