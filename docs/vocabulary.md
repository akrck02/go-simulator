# Vocabulary and conditions

Plugins never talk to each other, so they need to agree on names. If the terrain calls the height
of the ground `elevation` and the climate looks for `elevation`, they work together without knowing
the other exists. This page lists those shared names, their units, and the little language used
everywhere to say "when does this apply".

## Units

Positions and the sizes of shapes are in **world units**, the unit of the scene, on the ground
plane (x east, z south) with y up: they are what gets drawn. **Fields and properties are data**,
in the physical units written next to their names, so models read them without knowing the scale
of the scene. Plugins that need metres from world units take a `metres_per_unit` parameter (1000
for scenes in kilometres).

## Shared names

Defined in `world/vocabulary.go`. Names outside the vocabulary are allowed, but a name another
plugin may want to read belongs here.

**Fields**

| Name | Meaning | Unit | Written by |
|---|---|---|---|
| `elevation` | height of the ground | m | terrain |
| `slope` | steepness | rise over run | terrain |
| `water_depth` | water over the ground | m (0 = dry) | — |
| `temperature` | mean yearly temperature | °C | climate |
| `precipitation` | yearly rain and snow | mm | climate |
| `biome` | index into the names of the `biomes` event, −1 for the sea | — | climate |
| `vegetation` | index into the names of the `vegetation` event, −1 for bare ground | — | flora |
| `tree_cover`, `shrub_cover`, `herb_cover` | share of the ground under each layer | 0–1 | flora |

**Kinds**

`street` (a line with a width; `class` says which), `parcel` (a plot), `building` (a volume),
`wall`, `gate`, `square`, `bridge`, `river`, `tree` (its `species`), `anchor`.

**Properties**

`year` (when it appeared, in the stage's clock), `use` (`housing`, `shop`, `workshop`…),
`floors`, `class`, `sector`, `species`, `anchor` (true on anchors), `role` (what an anchor means).

**Events**

| Name | Data |
|---|---|
| `biomes` | `names`: the values of the `biome` field |
| `vegetation` | `names`, `labels`, `forms`, `heights`: the values of the `vegetation` field |

Events that name the categories of a field carry `{"names": […]}`.

## Conditions

"Buildings with three or more floors", "cells colder than 7 °C", "shops or workshops": many parts
of the system need to say when something applies, and they all use the same small language, in objectives, habitats, biome rules, managed
land and scene styles (package `conditions`). A condition is a map from a fact to what it must
be; every entry must hold.

| Written | Holds when the fact is |
|---|---|
| `kind: building` | equal to the value (numbers by value, the rest by text) |
| `use: [shop, workshop]` | one of the values |
| `floors: {min: 2, max: 3}` | a number in the range (either end may be left out) |
| `river: true` | that yes/no value |

A fact that does not exist never holds. For entities, the facts are `kind` and their properties;
for cells of a grid, the plugin says which facts it offers (the flora: `temperature`,
`precipitation`, `elevation`, `slope`, `biome` and `distance_to_<anchor>`). Rules files are
checked before a run: ranges take only `min` and `max`, and their bounds must be numbers.
