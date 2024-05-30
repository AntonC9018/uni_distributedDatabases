CREATE TABLE Client (
    id INT NOT NULL IDENTITY(1,1) PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    nume VARCHAR(255) NOT NULL,
    prenume VARCHAR(255) NOT NULL);

CREATE TABLE Foaie (
    id INT NOT NULL IDENTITY(1,1) PRIMARY KEY,
    tip VARCHAR(10) NOT NULL,
    pret FLOAT NOT NULL,
    providedTransport BIT NOT NULL,
    hotel VARCHAR(255) NOT NULL);

CREATE TABLE Rezervare (
    ordNum INT NOT NULL IDENTITY(1,1) PRIMARY KEY,
    clientId INT NOT NULL,
    foaieId INT NOT NULL,
    dataRezervarii DATE NOT NULL,
    gaj FLOAT NOT NULL);

CREATE TABLE Cumparatura (
    ordNum INT NOT NULL IDENTITY(1,1) PRIMARY KEY,
    clientId INT NOT NULL,
    foaieId INT NOT NULL,
    dataCumpararii DATE NOT NULL);   

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

INSERT INTO Client (email, nume, prenume) VALUES ('anton@email.com', 'Anton', 'Ivan');
INSERT INTO CLIENT (email, nume, prenume) VALUES ('ussr@gmail.com', 'Hello', 'Russia');
INSERT INTO CLIENT (email, nume, prenume) VALUES ('gg@gg.ru', 'Fella', 'Johns');

INSERT INTO Foaie (tip, pret, providedTransport, hotel) VALUES ('Munte', 100.00, 1, 'Maldivi');
INSERT INTO Foaie (tip, pret, providedTransport, hotel) VALUES ('Mare', 150.00, 0, '5 star');
INSERT INTO Foaie (tip, pret, providedTransport, hotel) VALUES ('Excursie', 200.00, 1, 'Beste');

INSERT INTO Rezervare (clientId, foaieId, dataRezervarii, gaj) VALUES (1, 1, '2023-01-15', 50.00);
INSERT INTO Rezervare (clientId, foaieId, dataRezervarii, gaj) VALUES (2, 2, '2023-02-20', 75.00);
INSERT INTO Rezervare (clientId, foaieId, dataRezervarii, gaj) VALUES (3, 3, '2023-03-25', 100.00);

INSERT INTO Cumparatura (clientId, foaieId, dataCumpararii) VALUES (1, 1, '2023-01-10');
INSERT INTO Cumparatura (clientId, foaieId, dataCumpararii) VALUES (2, 2, '2023-02-15');
INSERT INTO Cumparatura (clientId, foaieId, dataCumpararii) VALUES (3, 3, '2023-03-20');

CREATE PARTITION FUNCTION [tipPartitionFunc](VARCHAR(10))
AS RANGE LEFT FOR VALUES ('Mare', 'Munte', 'Excursie');

CREATE PARTITION SCHEME [tipPartitionScheme]
AS PARTITION [tipPartitionFunc]
ALL TO ([PRIMARY]);

-- 1. Ștergem constrângerile de chei străine către Foaie.
ALTER TABLE Rezervare DROP CONSTRAINT fk_Rezervare_foaieId;
ALTER TABLE Cumparatura DROP CONSTRAINT fk_Cumparatura_foaieId;

-- 2. Renumim Foaie
exec sp_rename 'Foaie', 'Foaie_old';

-- 3. Creăm Foaie partiționat
CREATE TABLE Foaie (
    id INT NOT NULL IDENTITY(1, 1),
    pret FLOAT NOT NULL,
    providedTransport BIT NOT NULL,
    hotel VARCHAR(255) NOT NULL,
    tip VARCHAR(10) NOT NULL,
    PRIMARY KEY (id, tip)
) ON tipPartitionScheme(tip);

-- Stop it from trying to generate the ids automatically:
SET IDENTITY_INSERT Foaie ON;

-- 4. Copiem datele în partiții
INSERT INTO Foaie (id, pret, providedTransport, hotel, tip)
SELECT id, pret, providedTransport, hotel, tip
FROM Foaie_old;

-- 5. Ștergem Foaie veche
DROP TABLE Foaie_old;
