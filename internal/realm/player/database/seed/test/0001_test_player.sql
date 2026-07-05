--liquibase formatted sql

--changeset pixels:pixels-player-seed-test-0001-test-player
insert into players (id, username)
overriding system value
values (2, 'test_player')
on conflict do nothing;

insert into player_profiles (player_id, look, gender, motto)
values (2, 'hr-100.hd-180-1.ch-210-66.lg-270-82.sh-290-80', 'M', 'Test fixture.')
on conflict do nothing;
--rollback delete from player_profiles where player_id = 2;
--rollback delete from players where id = 2;
