CREATE TABLE IF NOT EXISTS items (
    id SERIAL PRIMARY KEY,
    naming varchar(255) NOT NULL,
    weight FLOAT NOT NULL,
    description varchar(255) NOT NULL
);