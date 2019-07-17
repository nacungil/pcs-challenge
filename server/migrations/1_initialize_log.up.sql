CREATE TABLE logs (
    id serial PRIMARY KEY,
    peer_id serial,
    action varchar(20),
    created_on TIMESTAMP NOT NULL
);

CREATE TABLE results (
    id serial PRIMARY KEY,
    peer_id serial,
    res varchar(200),
    created_on TIMESTAMP NOT NULL
);

CREATE TABLE jobs (
    id SERIAL PRIMARY KEY
);
