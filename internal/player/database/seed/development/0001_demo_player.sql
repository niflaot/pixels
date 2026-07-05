--liquibase formatted sql

--changeset pixels:pixels-player-seed-development-0001-demo-player
insert into players (id, username)
overriding system value
values (1, 'demo')
on conflict do nothing;

insert into player_profiles (player_id, look, gender, motto)
values (1, 'hr-100.hd-180-1.ch-210-66.lg-270-82.sh-290-80', 'M', 'Welcome to Pixels.')
on conflict do nothing;
--rollback delete from player_profiles where player_id = 1;
--rollback delete from players where id = 1;
