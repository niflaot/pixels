--liquibase formatted sql

--changeset pixels:furniture-seed-curiosities-placement-0045 context:development
update furniture_items
set x=5,y=2,updated_at=now()
where id=970101 and room_id=160 and (x is distinct from 5 or y is distinct from 2);
--rollback update furniture_items set x=2,y=2,updated_at=now() where id=970101 and room_id=160;
