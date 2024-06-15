DO $$
DECLARE
    max_id INT;
BEGIN
    SELECT COALESCE(MAX(id), 0) INTO max_id FROM Foaie;

    FOR i IN 1..200 LOOP
        INSERT INTO Foaie (id, tip, pret, providedTransport, hotel)
        VALUES (
			max_id + i,
            (ARRAY['Munte', 'Mare', 'Excursie'])[floor(random() * 3 + 1)]::FoaieTip,
            round(cast(random() * 9 + 1 as numeric), 1) * 100,
            (random() > 0.5)::boolean,
            (ARRAY['Hotel Transylvania', 'Sea Breeze Hotel', 'Mountain Retreat', 'City Explorer', 'Sunset Resort'])[floor(random() * 5 + 1)]
        );
    END LOOP;
END $$;
