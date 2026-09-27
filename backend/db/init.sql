CREATE TABLE IF NOT EXISTS players (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    position SMALLINT NOT NULL
);

INSERT INTO players (name, position) VALUES ('Ernesto Provitillo', 5);
INSERT INTO players (name, position) VALUES ('Alfre Montes de Oca', 1);
