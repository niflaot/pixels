--liquibase formatted sql

--changeset pixels:pixels-furniture-0016-validate-room-footprints splitStatements:false
create or replace function pixels_furniture_footprint_fits_heightmap(
    target_heightmap text,
    target_x integer,
    target_y integer,
    definition_width integer,
    definition_length integer,
    target_rotation integer
) returns boolean
language plpgsql
immutable
as $$
declare
    normalized text;
    rows text[];
    footprint_width integer;
    footprint_length integer;
    offset_x integer;
    offset_y integer;
    tile text;
begin
    if target_heightmap is null
        or target_x < 0
        or target_y < 0
        or definition_width <= 0
        or definition_length <= 0
        or target_rotation not in (0,2,4,6) then
        return false;
    end if;

    normalized := replace(replace(btrim(target_heightmap), E'\r\n', E'\n'), E'\r', E'\n');
    rows := string_to_array(normalized, E'\n');
    if target_rotation in (2,6) then
        footprint_width := definition_length;
        footprint_length := definition_width;
    else
        footprint_width := definition_width;
        footprint_length := definition_length;
    end if;

    for offset_y in 0..footprint_length-1 loop
        if target_y + offset_y + 1 > cardinality(rows) then
            return false;
        end if;
        for offset_x in 0..footprint_width-1 loop
            if target_x + offset_x + 1 > char_length(rows[target_y + offset_y + 1]) then
                return false;
            end if;
            tile := substr(rows[target_y + offset_y + 1], target_x + offset_x + 1, 1);
            if lower(tile) = 'x' or tile !~ '^[0-9A-Za-z]$' then
                return false;
            end if;
        end loop;
    end loop;

    return true;
end;
$$;

create or replace function pixels_furniture_item_footprint_fits(
    target_room_id bigint,
    target_definition_id bigint,
    target_x integer,
    target_y integer,
    target_rotation integer
) returns boolean
language plpgsql
stable
as $$
declare
    target_heightmap text;
    definition_width integer;
    definition_length integer;
begin
    select coalesce(custom.heightmap, fixed.heightmap), definition.width, definition.length
    into target_heightmap, definition_width, definition_length
    from rooms room
    join furniture_definitions definition
      on definition.id = target_definition_id
     and definition.kind = 'floor'
     and definition.deleted_at is null
    left join room_custom_layouts custom on custom.room_id = room.id
    left join room_layouts fixed
      on fixed.name = room.model_name
     and fixed.deleted_at is null
    where room.id = target_room_id
      and room.deleted_at is null;

    return pixels_furniture_footprint_fits_heightmap(
        target_heightmap,
        target_x,
        target_y,
        definition_width,
        definition_length,
        target_rotation
    );
end;
$$;

create or replace function pixels_assert_room_furniture_footprints(
    target_room_id bigint,
    target_heightmap text
) returns void
language plpgsql
as $$
declare
    invalid_item_id bigint;
begin
    select item.id
    into invalid_item_id
    from furniture_items item
    join furniture_definitions definition on definition.id = item.definition_id
    where item.room_id = target_room_id
      and item.wall_position is null
      and item.deleted_at is null
      and pixels_furniture_footprint_fits_heightmap(
          target_heightmap,
          item.x,
          item.y,
          definition.width,
          definition.length,
          item.rotation
      ) is not true
    order by item.id
    limit 1;

    if invalid_item_id is not null then
        raise exception 'furniture item % has a footprint outside room %', invalid_item_id, target_room_id
            using errcode = '23514', constraint = 'furniture_items_room_footprint_chk';
    end if;
end;
$$;

create or replace function pixels_guard_furniture_item_footprint()
returns trigger
language plpgsql
as $$
begin
    if new.room_id is not null
        and new.wall_position is null
        and new.deleted_at is null
        and pixels_furniture_item_footprint_fits(
            new.room_id,
            new.definition_id,
            new.x,
            new.y,
            new.rotation
        ) is not true then
        raise exception 'furniture item % has a footprint outside room %', new.id, new.room_id
            using errcode = '23514', constraint = 'furniture_items_room_footprint_chk';
    end if;

    return new;
end;
$$;

create or replace function pixels_guard_custom_layout_furniture()
returns trigger
language plpgsql
as $$
declare
    fallback_heightmap text;
begin
    if tg_op = 'DELETE' then
        select fixed.heightmap
        into fallback_heightmap
        from rooms room
        join room_layouts fixed
          on fixed.name = room.model_name
         and fixed.deleted_at is null
        where room.id = old.room_id;
        perform pixels_assert_room_furniture_footprints(old.room_id, fallback_heightmap);
        return old;
    end if;

    perform pixels_assert_room_furniture_footprints(new.room_id, new.heightmap);
    return new;
end;
$$;

create or replace function pixels_guard_fixed_layout_furniture()
returns trigger
language plpgsql
as $$
declare
    invalid_item_id bigint;
    invalid_room_id bigint;
begin
    select item.id, item.room_id
    into invalid_item_id, invalid_room_id
    from furniture_items item
    join rooms room on room.id = item.room_id
    join furniture_definitions definition on definition.id = item.definition_id
    left join room_custom_layouts custom on custom.room_id = room.id
    where room.model_name = old.name
      and custom.room_id is null
      and item.wall_position is null
      and item.deleted_at is null
      and pixels_furniture_footprint_fits_heightmap(
          new.heightmap,
          item.x,
          item.y,
          definition.width,
          definition.length,
          item.rotation
      ) is not true
    order by item.id
    limit 1;

    if invalid_item_id is not null then
        raise exception 'layout % would invalidate furniture item % in room %', old.name, invalid_item_id, invalid_room_id
            using errcode = '23514', constraint = 'furniture_items_room_footprint_chk';
    end if;

    return new;
end;
$$;

create or replace function pixels_guard_room_model_furniture()
returns trigger
language plpgsql
as $$
declare
    target_heightmap text;
begin
    if exists (select 1 from room_custom_layouts where room_id = new.id) then
        return new;
    end if;

    select heightmap
    into target_heightmap
    from room_layouts
    where name = new.model_name
      and deleted_at is null;
    perform pixels_assert_room_furniture_footprints(new.id, target_heightmap);
    return new;
end;
$$;

create or replace function pixels_guard_definition_footprints()
returns trigger
language plpgsql
as $$
declare
    invalid_item_id bigint;
    invalid_room_id bigint;
begin
    select item.id, item.room_id
    into invalid_item_id, invalid_room_id
    from furniture_items item
    join rooms room on room.id = item.room_id
    left join room_custom_layouts custom on custom.room_id = room.id
    left join room_layouts fixed
      on fixed.name = room.model_name
     and fixed.deleted_at is null
    where item.definition_id = new.id
      and item.wall_position is null
      and item.deleted_at is null
      and pixels_furniture_footprint_fits_heightmap(
          coalesce(custom.heightmap, fixed.heightmap),
          item.x,
          item.y,
          new.width,
          new.length,
          item.rotation
      ) is not true
    order by item.id
    limit 1;

    if invalid_item_id is not null then
        raise exception 'definition % would invalidate furniture item % in room %', new.id, invalid_item_id, invalid_room_id
            using errcode = '23514', constraint = 'furniture_items_room_footprint_chk';
    end if;

    return new;
end;
$$;

update furniture_items
set x = 5,
    y = 2,
    updated_at = now(),
    version = version + 1
where id = 970101
  and room_id = 160
  and (x is distinct from 5 or y is distinct from 2);

do $$
declare
    invalid_item_id bigint;
begin
    select item.id
    into invalid_item_id
    from furniture_items item
    where item.room_id is not null
      and item.wall_position is null
      and item.deleted_at is null
      and pixels_furniture_item_footprint_fits(
          item.room_id,
          item.definition_id,
          item.x,
          item.y,
          item.rotation
      ) is not true
    order by item.id
    limit 1;

    if invalid_item_id is not null then
        raise exception 'existing furniture item % has an invalid room footprint', invalid_item_id
            using errcode = '23514', constraint = 'furniture_items_room_footprint_chk';
    end if;
end;
$$;

create trigger furniture_items_room_footprint_guard
before insert or update of room_id,definition_id,x,y,rotation,wall_position,deleted_at
on furniture_items
for each row
execute function pixels_guard_furniture_item_footprint();

create trigger room_custom_layouts_furniture_guard
before insert or update of heightmap or delete
on room_custom_layouts
for each row
execute function pixels_guard_custom_layout_furniture();

create trigger room_layouts_furniture_guard
before update of heightmap
on room_layouts
for each row
execute function pixels_guard_fixed_layout_furniture();

create trigger rooms_model_furniture_guard
before update of model_name
on rooms
for each row
execute function pixels_guard_room_model_furniture();

create trigger furniture_definitions_footprint_guard
before update of width,length
on furniture_definitions
for each row
execute function pixels_guard_definition_footprints();

--rollback drop trigger if exists furniture_definitions_footprint_guard on furniture_definitions;
--rollback drop trigger if exists rooms_model_furniture_guard on rooms;
--rollback drop trigger if exists room_layouts_furniture_guard on room_layouts;
--rollback drop trigger if exists room_custom_layouts_furniture_guard on room_custom_layouts;
--rollback drop trigger if exists furniture_items_room_footprint_guard on furniture_items;
--rollback drop function if exists pixels_guard_definition_footprints();
--rollback drop function if exists pixels_guard_room_model_furniture();
--rollback drop function if exists pixels_guard_fixed_layout_furniture();
--rollback drop function if exists pixels_guard_custom_layout_furniture();
--rollback drop function if exists pixels_guard_furniture_item_footprint();
--rollback drop function if exists pixels_assert_room_furniture_footprints(bigint,text);
--rollback drop function if exists pixels_furniture_item_footprint_fits(bigint,bigint,integer,integer,integer);
--rollback drop function if exists pixels_furniture_footprint_fits_heightmap(text,integer,integer,integer,integer,integer);
