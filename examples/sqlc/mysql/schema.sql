create table if not exists authors
(
    id   bigint       not null auto_increment primary key,
    name varchar(512) not null,
    bio  text
);
