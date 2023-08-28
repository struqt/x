create table if not exists authors
(
    id   bigserial primary key,
    name varchar not null,
    bio  text
);
