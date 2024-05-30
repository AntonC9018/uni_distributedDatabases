CREATE TABLE Client (
    id SERIAL PRIMARY KEY NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    nume VARCHAR(255) NOT NULL,
    prenume VARCHAR(255) NOT NULL
);

CREATE TYPE FoaieTip AS ENUM ('Munte', 'Mare', 'Excursie');

CREATE TABLE Foaie (
    id SERIAL PRIMARY KEY NOT NULL,
    tip FoaieTip NOT NULL,
    pret MONEY NOT NULL,
    providedTransport BOOLEAN NOT NULL,
    hotel VARCHAR(255) NOT NULL
);

CREATE TABLE Rezervare (
    ordNum SERIAL PRIMARY KEY NOT NULL,
    clientId INT REFERENCES Client(id) NOT NULL,
    foaieId INT REFERENCES Foaie(id) NOT NULL,
    dataRezervarii DATE NOT NULL,
    gaj MONEY NOT NULL
);

CREATE TABLE Cumparatura (
    ordNum SERIAL PRIMARY KEY NOT NULL,
    clientId INT REFERENCES Client(id) NOT NULL,
    foaieId INT REFERENCES Foaie(id) NOT NULL,
    dataCumpararii DATE NOT NULL
);

ALTER TABLE Rezervare
    ADD CONSTRAINT fk_Rezervare_clientId
    FOREIGN KEY (clientId)
    REFERENCES Client(id);

ALTER TABLE Rezervare
    ADD CONSTRAINT fk_Rezervare_foaieId
    FOREIGN KEY (foaieId)
    REFERENCES Foaie(id);

ALTER TABLE Cumparatura
    ADD CONSTRAINT fk_Cumparatura_clientId
    FOREIGN KEY (clientId)
    REFERENCES Client(id);

ALTER TABLE Cumparatura
    ADD CONSTRAINT fk_Cumparatura_foaieId
    FOREIGN KEY (foaieId)
    REFERENCES Foaie(id);

INSERT INTO Client (email, nume, prenume) VALUES
    ('test1@example.com', 'Doe', 'John'),
    ('test2@example.com', 'Smith', 'Jane'),
    ('test3@example.com', 'Johnson', 'Bob');

INSERT INTO Foaie (tip, pret, providedTransport, hotel) VALUES
    ('Munte', 100.00, true, 'Hotel A'),
    ('Mare', 150.00, false, 'Hotel B'),
    ('Excursie', 200.00, true, 'Hotel C');

INSERT INTO Rezervare (clientId, foaieId, dataRezervarii, gaj) VALUES
    (1, 1, TO_DATE('2023-01-15', 'YYYY-MM-DD'), 50.00),
    (2, 2, TO_DATE('2023-02-20', 'YYYY-MM-DD'), 75.00),
    (3, 3, TO_DATE('2023-03-25', 'YYYY-MM-DD'), 100.00);

INSERT INTO Cumparatura (clientId, foaieId, dataCumpararii) VALUES
    (1, 1, TO_DATE('2023-01-10', 'YYYY-MM-DD')),
    (2, 2, TO_DATE('2023-02-15', 'YYYY-MM-DD')),
    (3, 3, TO_DATE('2023-03-20', 'YYYY-MM-DD'));

ALTER TABLE Foaie RENAME TO Foaie_old;

CREATE TABLE Foaie (
    id SERIAL PRIMARY KEY NOT NULL,
    pret MONEY NOT NULL,
    providedTransport BOOLEAN NOT NULL,
    hotel VARCHAR(255) NOT NULL,
    tip FoaieTip NOT NULL
) PARTITION BY LIST (tip);

ALTER TABLE Rezervare
    DROP CONSTRAINT fk_Rezervare_foaieId;
ALTER TABLE Cumparatura
    DROP CONSTRAINT fk_Cumparatura_foaieId;
ALTER TABLE Rezervare
    DROP CONSTRAINT rezervare_foaieid_fkey;
ALTER TABLE Cumparatura
    DROP CONSTRAINT cumparatura_foaieid_fkey;

CREATE TABLE Foaie_Munte PARTITION OF Foaie FOR VALUES IN ('Munte');
CREATE TABLE Foaie_Mare PARTITION OF Foaie FOR VALUES IN ('Mare');
CREATE TABLE Foaie_Excursie PARTITION OF Foaie FOR VALUES IN ('Excursie');

INSERT INTO Foaie_Munte (id, pret, providedTransport, hotel, tip)
    SELECT id, pret, providedTransport, hotel, tip FROM Foaie_old WHERE tip = 'Munte';
INSERT INTO Foaie_Mare (id, pret, providedTransport, hotel, tip)
    SELECT id, pret, providedTransport, hotel, tip FROM Foaie_old WHERE tip = 'Mare';
INSERT INTO Foaie_Excursie (id, pret, providedTransport, hotel, tip)
    SELECT id, pret, providedTransport, hotel, tip FROM Foaie_old WHERE tip = 'Excursie';
  
DROP TABLE Foaie_old;

ALTER TABLE Rezervare
    ADD CONSTRAINT fk_Rezervare_foaieId
    FOREIGN KEY (foaieId)
    REFERENCES Foaie(id);

ALTER TABLE Cumparatura
    ADD CONSTRAINT fk_Cumparatura_foaieId
    FOREIGN KEY (foaieId)
    REFERENCES Foaie(id);

CREATE TABLE Foaie (
    id SERIAL NOT NULL,
    pret MONEY NOT NULL,
    providedTransport BOOLEAN NOT NULL,
    hotel VARCHAR(255) NOT NULL,
    tip FoaieTip NOT NULL,
    PRIMARY KEY (id, tip)
) PARTITION BY LIST (tip);
