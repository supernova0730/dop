drop table if exists t1 cascade;

create table t1
(
    c1 int,
    c2 int[],
    c3 jsonb,
    c4 text
);

insert into t1 (c1, c2, c3, c4)
values (1, array [1, 2, 3, 4], '{
  "a": 1,
  "b": [
    1,
    2
  ],
  "c": "asd"
}', '123'),
       (2, array [4,3,1], '{
         "a": 4,
         "b": [
           8
         ],
         "c": "iii"
       }', 'poi');
