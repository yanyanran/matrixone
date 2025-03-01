drop table if exists t1;
drop table if exists t2;
create table t1(a int, b int);
insert into t1 values (1, 10),(2, 20);
create table t2(aa int, bb varchar(20));
insert into t2 values (11, "aa"),(22, "bb");

(select a from t1) order by b desc;
(((select a from t1) order by b desc));
(((select a from t1))) order by b desc;
(((select a from t1 order by b desc) limit 1));
(((select a from t1 order by b desc))) limit 1;
(select a from t1 union select aa from t2) order by a desc;

(select a from t1 order by a) order by a;
(((select a from t1 order by a))) order by a;
(((select a from t1) order by a)) order by a;
(select a from t1 limit 1) limit 1;
(((select a from t1 limit 1))) limit 1;
(((select a from t1) limit 1)) limit 1;
(select a from t1 union select aa from t2) order by aa;
select a from t1 union select a from t1;
drop table if exists t3;
create table t3(
a tinyint
);
insert into t3 values (20),(10),(30),(-10);
drop table if exists t4;
create table t4(
col1 smallint,
col2 smallint unsigned,
col3 float,
col4 bool
);
insert into t4 values(100, 65535, 127.0, 1);
insert into t4 values(300, 0, 1.0, 0);
insert into t4 values(500, 100, 0.0, 0);
insert into t4 values(200, 35, 127.0, 1);
insert into t4 values(200, 35, 127.44, 1);
select a from t3 union select col3 from t4 order by a;
drop table if exists t7;
CREATE TABLE t7 (
a int not null,
b char (10) not null
);
insert into t7 values(1,'a'),(2,'b'),(3,'c'),(3,'c');
select * from (select * from t7 union all select * from t7 limit 2) a;
drop table if exists t2;
CREATE TABLE t2 ( id int(3) unsigned default '0', id_master int(5) default '0', text1 varchar(5) default NULL, text2 varchar(5) default NULL);
INSERT INTO t2 (id, id_master, text1, text2) VALUES("1", "1", "foo1", "bar1");
INSERT INTO t2 (id, id_master, text1, text2) VALUES("2", "1", "foo2", "bar2");
INSERT INTO t2 (id, id_master, text1, text2) VALUES("3", "1", NULL, "bar3");
INSERT INTO t2 (id, id_master, text1, text2) VALUES("4", "1", "foo4", "bar4");
SELECT 1 AS id_master, 1 AS id, NULL AS text1, 'ABCDE' AS text2 UNION SELECT id_master, id, text1, text2 FROM t2 order by id;