# Niebla

## Pitch
An isometric survival game about the Gray Fog: a planet-covering mist of self-replicating nanites that digests anything that stands still. You run a small colony of robots around an indestructible fog-repelling core: keep the bubble fed with oil, harvest lilac mineral outside it, and grow, one repulsor at a time. Other people of the mist arrive, establish mining cities and send forces from them for the same oil. One state, one truth: the whole game is a serializable simulation that can be saved, replayed and later served to many players.

## Core loop
The fog breathes in cycles: every so often it swells, pressing its line in, then recedes; the core's permanent bubble is free, while each colony protector needs its own oil. Outside the bubbles, oil pools and lilac veins wait. Every cycle:
1. Keep protectors supplied and robots charged; save spendable oil for building and war.
2. Send charged robots out to harvest oil and lilac, and back before their charge runs out - a robot that stalls in the fog is digested.
3. Mark build jobs (protectors, pumps and their pipes, silos); robots build them.
4. A real protector widens or holds the safe ground, but it drinks oil from a dedicated tank. Robots or pipes can refill it; when its tank gets low, its radius fades to nothing. The core is free of tension; the need for resources outside is what pushes you out.
5. Spend on war what the rivals make you spend: a guard post near a
   threatened tank, squads against a city sortie, mechanics to restore
   buildings, and artillery against mobile artillery. Oil is every shot
   and every repair, lilac every shell, so war and growth draw on the
   same stores (see [The rivals](#the-rivals)).

## Resources
| Resource | What it is | What it is for |
| --- | --- | --- |
| Oil | Pumped from pools | Dedicated protector fuel; robot charge ("food"); every shot of a guard post and of a trooper; mechanic repairs; a part of every shell; what the rivals come to steal |
| Lilac mineral | Mined from veins (StarCraft vespene / Dune spice vibe) | Every construction and upgrade; troopers and mechanics; every artillery shell |

There is no food and no people in the colony: the colonists are robots, and an uncharged robot outside a bubble dies to the fog. The only people in the region so far are the rivals, and they ride.

## The core
An indestructible repelling core, deployed at the start of a region. It is the learning zone: a small, permanent bubble with no upkeep, so the first cycles are comfortable. Nothing hurts it, including rival shells; scouts and raiders can still siphon its tank. It cannot win the game: its bubble is small, and everything the colony needs lies outside. Growth has a price from the moment you leave.

## Shadow protectors
Every protector has a dedicated oil tank, separate from the colony's spendable stores. Its initial charge is included in its blueprint cost. The protector burns oil continuously; a pipe or a robot hauling oil can replenish it. Its bubble holds full size above the adjustable low-fuel threshold, then shrinks linearly to zero as the remaining charge runs out. An empty protector stays built but shelters nothing. Existing saves receive the initial charge once when they migrate to this rule.

## The fog
The fog is law outside the bubbles: it drags at every walker's pace and digests whatever stands still in it on an empty tank. It never damages a bubble directly: the core's repulsion is absolute, while a protector's shrinking radius reflects only its own oil supply.

The fog does not creep toward the core, and the colony never maintains a wall of repulsors: conflict is temporal, not positional. The fog **breathes**: every `fogSwellPeriod` cycles a **swell** rises - the line presses in `fogSwellReach` tiles for `fogSwellTicks`, then recedes. A swell is telegraphed a cycle ahead (`swell in N cycles` in the HUD), so against the fog the game is deciding *when* to go out, never where to stack turrets (the people in it are another matter: see [The rivals](#the-rivals)): the answer to a swell is timing - come home, or sit it out in an outpost's bubble, which makes protectors and chargers storm shelters rather than guns. Inside a swell the mist is meaner: the drag deepens (`fogSpeedFactor` 0.5 to 0.25 in the pushed band) and tanks burn 1.5x, so a loaded robot caught mid-haul pays. Difficulty scales with the swells - reach, length, how often they come (`fogSwellGrowth` per cycle) - never in ground lost for good, and the first hour of real time can pass with hardly a swell. The weather is state, not scenery: the fog's cycle counter and swell live in `State`, so a save reproduces its mists.

**Clear air shows as clear air.** Inside an active repulsor's circle - the core's, a protector's, a rival crawler's - the ground shows as it is, with no tint and no mist, wherever the circle stands, deep fog and swells included: a protector out in the mist is a round window onto its ground, shrinking as its oil runs low. Outside every circle a haze hangs over the clear ground (`mistHaze` 0.3), so the eye reads where the air is safe before it reads any ring, and past the fog's line the haze thickens to the fog the region is surrounded by. What a swell adds - the darker crests and the static - is the mist's own and stops at the circles too.

Where the core's and protectors' circles overlap, their clear ground joins: the outline shows only the outside edge of the combined area, with every inner arc hidden.

**A swell comes little by little.** It starts on its tick, at a cycle's end, and ends on its tick, but the ground feels it through the swell's **pressure** (`Fog.Pressure`, 0 to 1), which takes `fogSwellRampTicks` 300 (5 s) to come all the way in and as long to let go: the line travels instead of jumping, the pushed band widens with it, and the extra burn grows with it (1 to `fogSwellBurn`). The HUD says `swell` while it is up and `swell easing` while the line walks back out.

**A swell shows** (`swell.go`), as strongly as it presses: waves of shade are born far out in the mist (`swellWaveRim` 17 tiles), roll in at `swellWaveSpeed` 0.6 tiles a second, darken on the way and break on the line - `swellWaves` 7 crests, each wobbling and broken into arcs its own way, like wind combing snow - and the mist itself crawls with static, up to `swellStaticSpecks` 900 one-pixel specks on the screen that jump every `swellStaticTicks` 3 updates, only where fog stands and thicker where it is thicker, as if the mites were a field and not a swarm. Both are black over the fog, the mites' own ink, only fainter and everywhere. They are drawn from the state's tick and nothing else: no randomness, nothing kept between frames.

**The wear shows.** Whatever stands in the mist - a robot, a building site, a pile - wears mites of darkness: a few per cubic meter of its body (`mitesPerCubicUnit`, thinned by how much fog stands on it), each a spark turned inside out: black only at the heart (opacity 1), then a round halo that starts at `miteHaloOpacity` 0.2 beside it and eases down ring by ring to nothing at the rim (`miteFalloff`: 1, 0.2, 0.13, 0.07, 0.03, 0.01, then clear), painted over the fog in plain black so it takes light away instead of adding it. They orbit their host and chase that orbit late (`miteLagSeconds`), so a walker drags its swarm behind it, never quite caught, and the moment it stops - to load, to build, to wait - the orbit tightens onto the body (`miteGripSeconds`). It is the fog's law drawn: it digests what stands still. When the host is gone, digested or hauled away, its swarm closes on the empty spot and fades. **Mites** is the official name, the one a manual would print. The people who live with them call them something that can't be printed - the working candidate is *ballbusters* - which means the game owes itself people: not colonists to feed, but voices, the way Oxygen Not Included tells of duplicants who came before; how they show up (logs, wrecks, a radio) is open. The mites are view, not state (`mites.go`): nothing in the simulation reads them, their randomness is golib's, and a save knows nothing of them.

The fog is also the frontier, and going out into it is the robot's adventure (a pillar for later): beyond routine hauling, a robot can be sent *into* the mist on an expedition - deeper fog, richer finds, things a commuter never meets. The fog digests the careless; adventure is the reward for daring it well.

## Construction model
The player never builds by hand. Marking is building, and building is earned: the blueprints arrive at the core as **remote schematics** (`sim_tech.go`, the ladder and its triggers; the badge and the callout are view, `tech.go`). A drop lights a **badge over the monolith** - a plate ringed in the drop's ink around its group's mark or the blueprint's own body, two halos breathing with the state's tick - and the HUD adds `schematics at the core` while one waits, so a player far from the core knows they are called. The click on the badge opens the drop: the badge goes (`AckTech`, the only thing about a drop the state keeps - a save with the badge unclicked keeps it) and a **callout** by the core says what arrived, one line of what it is for and, under it, a **square per thing it brings** - the icon of its construction above and its name below, the way a deposit's card shows its workers; the pipes, which build no building, wear a mark of their own. Clicking the callout closes it; clicking elsewhere closes it and still acts on the region. Arrival itself is derived from the state, never stored: an old save wakes with exactly what it has earned, opened, so no pile of badges greets it. And **nothing unreachable is ever offered**: an option the schematics, the ground or the fog refuse is not on the rings at all - the menu offers only what the cell could really take, a group with nothing to raise doesn't appear, a ring that empties while it stands open puts the menu away, and before the first schematics the menu doesn't open at all. The stores are the one referee that takes nothing off the rings: a blueprint they can't pay stands washed out to gray, refuses the click with a dull click of its own, and the tip beside it names it, prices it and boxes in red each resource that falls short. `build pump` and `lay pipe` stay off their cards until they would work. `MarkBuilding` and `LayPipe` refuse what hasn't arrived, so a replay or a cheat builds nothing early. The ladder, and the saw it rides, is in [Introduction to the game](#introduction-to-the-game).

1. A click on a free cell of buildable ground opens the build menu right there, a radial around the cell. The grid's last subdivision is the cell, 25 u on a side (8 by 8 to a tile, about four robots across), the footprint of the smallest building - so a tile can grow a hamlet. A deposit's body is organic, and the ground under it obeys the eye: a cell its ore doesn't cover builds like any other (`oreAt`, `buildableGround`), and where the ore draws, only the pump may stand. The core starts with a gift of stores (`startingStockOil` 300 L, `startingStockLilac` 600 kg) so the first buildings need no haul first.
2. The menu is two rings deep: the first offers the **build groups** - **industry** (the factories), **military** and **logistics** - and the second the blueprints of the group picked (industry: factory, war factory; military: guard post, artillery; logistics: charger, silo, warehouse, protector). Only what the colony could raise on that very cell stands on the rings: what the schematics, the ground or the fog refuse is not offered. A blueprint the stores can't pay stands washed out to gray - its group with it while nothing inside could be paid - the tip beside it names it, prices it and boxes in red each resource that falls short, and a click on it refuses and the menu stays. Every option wears an icon: a group its own mark, a blueprint the very body the region draws, in miniature, so one graphic serves both. Picking a blueprint pays its cost and marks it on that very cell - there is no second pick: the click that opened the menu already chose where. A right click goes back a ring, and closes the menu from the first.
3. The job joins the queue; while robots raise it, the site shows a wireframe of the body to come with its progress bar, drawn over the fog.
4. Harvesting works the same way: mark an oil pool or a lilac vein, and robots commute between it and storage while their charge lasts.

Robots are simple units with priorities, not pathfinders of genius: the
player solves the layout, and the robots solve the walking. Builders and
workers are separate roles, so a construction job never pulls a worker away
from a deposit.

When a protector site is waiting, robots build it before any other
construction. Protectors keep their marked order, as do the remaining
jobs.

## Demolition and loose items
What is built can be unbuilt, and nothing is lost but the walking. (What a rival shell brings down goes the same way with half the refund: see [The rivals](#the-rivals).)

**The trash button.** The card of a building, and of a site still being raised, carries a small button with a trash can, at the right of the card's header. The core has none: it is indestructible both ways. The button asks twice: the first press arms it (the can turns red, the card says `demolish?`), the second demolishes, and a click anywhere else disarms it. Arming is view, not state; only the second press sends the action (`Demolish` for a building, `CancelJob` for a site).

**What demolishing does**, all in one action:
1. The building leaves the state at once - no work asked of the robots, no ruin left behind. A site leaves the job queue the same way.
2. Every task it had under way is cancelled: a factory's half-built robot never rolls out, a site's progress is gone. Robots carry no plan, so nobody has to be told: a builder finds no job the next tick, a robot refueling at a charger that is gone walks to the next nearest post.
3. Everything the building was made of or held falls to the ground where it stood, as one **pile**: its whole blueprint cost (`demolishRefund` 1.0, a dial), plus the cost of the robot a factory was building, plus whatever the stores no longer have a roof for - the stores are one stock under many roofs, so a silo or a warehouse "contains" the part of the stock that stops fitting when its roof goes, and that overflow leaves `State.Stock` and joins the pile. Nothing goes straight back to the stores: a refund is a haul.

**A protector is the one exception to the button**: it is dimmed while any other building or site stands under that protector's bubble alone, so the fog's law (nothing but a protector outside a bubble) can never be broken by taking one away. Demolish the outpost first, the protector last. Robots under it take their chances.

**The pile**, not a scatter. Loose items are one entity per demolished cell, a container drawn on the cell's middle that stands for everything lying there: a small heap of crates and drums, lilac or amber by what it holds, never under the dot size when far out, with a card of its own (`loose items`, its liters and kilograms, open by itself like any primary thing). It has no mass, no health and no capacity: it is bookkeeping with a picture, and the moment a robot takes the last of it, it is gone from the state. Scattering the items over the footprint was the other option and lost: a cell is 25 m across, so at any zoom but the closest a scatter reads as noise or as nothing, it multiplies entities and cards for no decision the player can take, and one pile keeps the state a small table (`State.Piles`, by ID: cell, oil, lilac). Until it is emptied a pile holds its cell - nothing can be marked there - and robots walk through it as they walk through everything. The fog leaves piles alone for now; whether it should nibble at what is left outside a bubble is an open dial, not a rule.

**Picking up.** Clearing piles is a line in the robot's day, after raising build jobs and before working its own post: what already lies on the ground comes home before anything new is dug. A robot with nothing better to do walks to the nearest pile that holds something the stores have room for, loads like at a deposit (`robotLoadTicks`, one kind per trip, lilac first, up to `robotCarryLilac` / `robotCarryOil`) and hauls it in. A pile the stores have no room for just waits - which is what happens to the overflow of a full silo torn down - so robots never stand around with their arms full because of it.

**Where a load goes.** The walk home ends at the nearest store of the cargo's kind: a warehouse or the core for lilac (solids), and for oil the nearest tank with room - a silo, a charger, a protector or the core - which is also where that oil then is (see [Oil has a place](#oil-has-a-place)). The core is a store like the others, and a robot unloads at its side (`storeStandoff` 11 u), not out at a parking spot. Lilac stays one stock and its roofs one sum; the nearest warehouse is only where the walking ends. A robot doesn't remember where its cargo came from, so this is one rule for every load, from a pile or from a deposit - which also makes silos and warehouses worth placing near the work, not just worth having.

## Oil has a place
Lilac is one stock under many roofs, but oil is not: it sits in **tanks** - the core's (`coreOilCap` 1000 L), each silo's (`siloOilCap` 1000 L), each charger's (`chargerOilCap` 200 L), and each protector's dedicated reserve. Oil gets between tanks in a robot's arms or down a pipe. The HUD and cards count oil available to spend; protector reserves are shown on their own cards and cannot pay colony costs or be stolen by rivals. Payments come from the available tanks, the core's first; but where the oil *is* now matters:
- A robot carries its oil to the nearest tank **with room**, including a protector's, and pours it into that one. Hauling stays what it was - slow, thirsty, exposed to the fog - and is how a colony with no pipes lives.
- A robot refills at the nearest charger or the core **with oil in it**, from that tank alone. A dry charger serves nobody until a robot or a pipe fills it.
- A demolished tank drops the oil it held on its cell, with its cost.

This is what makes pipes infrastructure rather than a shortcut: a silo beside a far pool holds that pool's oil, and it is a pipe, or the robots' legs, that brings it home.

## Pumps and pipes
The game has no belts and never will, but oil flows through pipes, so it stops riding in robots' arms where a pipe runs.

**The pump comes first.** A pump is a building that stands on an oil pool instead of on ground: the pool's card carries a `build pump` button (its cost beside it) while the stores can pay and its schematics have arrived, and the pump rises on the pool's middle like any site. A pool takes one pump, a dry pool none. A far pool can take an unprotected pump, but mites gather on the finished pump and digest it in 10 seconds without a protector's active bubble; a news plate explains the loss and half its cost falls as a pile. A protector raised in time halts the damage. A pump with no pipe does nothing, and a pool keeps its robot post, pump or not.

**A network, by ports.** A pipe carries oil one way, from a pump or a tank
into a tank: pool to silo, silo to another silo, silo to a charger or
protector, anything to the core or out of it. Buildings are the junctions:
each takes `pipePorts` 3 pipes, in and out together (the core
`corePipePorts` 6), and two ends take one pipe between them. Multiplexers
and mixers were the other option and wait: ports on the buildings that
already exist ask for no new building, and a silo is already a mixer with a
buffer. A pipe moves at most `pipeLitersPerSecond` 4 L/s, limited by its
source and its end's room. A source divides its supply equally among
outlets that can accept oil: a silo with two open pipes feeds both, and a
pump divides its `pumpLitersPerSecond` 2 L/s among its pipes. A
protector fills its own tank before passing oil onward. Once full, it
retains `protectorOilPerSecond` 0.25 L/s to stay powered and shares the
surplus equally among its outlets. A full protector without an outlet
receives only its upkeep; nothing is lost.

**Laying a pipe** is drawing it. The card of a pump, a silo, a charger, a protector or the core carries `lay pipe` while it has a port free, which arms the pointer: every left click on the ground adds a **bend**, a click on a tank - anywhere on its body; every one the pipe may end at wears a ring, and the label by the pointer says `to silo` before the click - ends the pipe there and marks it, and a right click takes the last bend back, or puts the pointer away when there is none. A building within reach wins over the core beside it, whose monolith is tall and would steal the click. A click on the pipe's **last node** (its last bend, or its source while it has none) opens a small menu around it: `connect`, which ends the pipe at the tank nearest that node, the curve, the price and the tank's name showing it before the pick; `undo`, which takes the node back; and `cancel`, which drops the pipe in hand. The pipe is a curve through the clicks - a centripetal Catmull-Rom spline, which passes through every bend and never loops between a short span and a long one - and with no bends at all it sags a little to one side instead of running like a ruler's line.

**The price is by the section** (`pipeSectionMeters` 25 m, a cell's side; `pipeSectionLilac` 5 kg apiece), paid when the pipe is marked. Marking is building here too, and it is work for many hands: a robot **claims a section** - of the oldest unlaid pipe's sections nobody else holds, the nearest to it -, tells the others by the claim itself, which is state (`Robot.Pipe`, `Robot.Section`), walks to the section's middle and **stands by it** for two seconds of work (`pipeSectionWorkTicks` 120), then claims another. So a pipe is laid in patches, by as many robots as it has free sections, the laid sections standing on their posts among the ghost of the rest, and the robots a pipe has no section for go on with their day. A claim lasts while the robot's task is the build line and dies with the robot, so a section is never orphaned. A pipe through the mist costs what walking and standing in the mist costs. Laying pipe is part of the build line of the robot's day, after the sites.

**It shows.** A pipe runs above the ground (`pipeLiftUnits` 9 m, never
under 5 px), on posts a section apart, and casts its shadow on the ground,
leaning away from the sun the buildings' faces imply. Its orange bands show
what the source offers: at the pump's full `pumpLitersPerSecond` 2 L/s,
orange fills 90% of each gap and leaves 10% steel gray to show motion. A
band moves by the liters actually transferred, so a sink that accepts less
slows it and a blocked or dry pipe stays gray. A protector fills before
passing oil; after that, it keeps 0.25 L/s for upkeep before sharing the
rest onward. Each protector in a chain narrows the bands by its upkeep.
Their phase follows the pipe's accumulated liters and survives saves.
Pipes may cross anything, the fog too, and are drawn over it like the sites.

**Taking it up.** The card of either end lists its pipes, each with where it goes or comes from, its length, what it is doing, and a `remove` button; demolishing a building takes its pipes with it. Either way a pipe's whole cost falls as a pile - by the building it started at, or on the demolished cell - and comes home as a haul.

Open: whether mites wear a pipe in the mist, junction buildings (a splitter with no tank) if ports prove short, priorities between a source's pipes, and whether a pipe needs upkeep.

## The rivals
The scout is the first contact: four minutes in it steals up to 25 L, paints its mark and leaves. The next visit is a simple crawler-led raiding party. It enters on the scout's saved bearing, camps, gives the player time to respond, steals from the nearest oil tank and leaves on that route. This is the last independently scheduled raid.

Thirty fog cycles (15 minutes) after that party leaves, a crawler arrives from a new deterministic bearing and drives to a settlement site 10 tiles (2 km) from the core, beyond the colony artillery's 1.5 km range. It establishes a city there, with the crawler serving as its construction rig. The Nexus comes after the pylon. Later cities arrive no sooner than 30 cycles apart, up to three; each city is part of the saved state. The city has a fixed 170 m repulsor post. Its first sortie needs no mobile artillery in calm weather, while the later artillery shields a whole battalion when the fog swells. The colony is told when a crawler approaches and when the city establishes.

For one minute after a city establishes, the HUD announces its location and
status. Its founding report and offscreen arrow use that same deadline,
unless newer news replaces them; other reports and arrows last
`reportShowTicks` 15 s.

The arrival crawler stays as the city's construction rig. The Nexus ID is reserved at founding, but its entity appears only after the antimist pylon is built. It has no repulsor of its own; the pylon covers it and the rest of the city. The extractor, lilac mine and military factory follow the Nexus, each a gray rival structure with health and a `cityBuildTicks` 90 s build. Oil and lilac are finite city-local reserves (`cityOilReserve` 900 L and `cityLilacReserve` 1800 kg). The extractors add to city stores; the factory spends them on sorties. Destroying an extractor stops that resource. Destroying the factory stops new forces; destroying the Nexus removes the city. City buildings never fire.

Once the factory and stores are ready, the first battalion assembles at the city and waits visibly for `campPrepareTicks`. It has raiders but no mobile artillery, walks directly to the nearest colony oil tank, siphons and returns to its city. The city waits `citySortieTicks` 5 min after a force returns, and never fields two sorties at once. Only one moving party can be in the region at a time; other cities wait their turn. The second battalion and later ones include mobile artillery: it carries its own 170 m antimist bubble and shells colony buildings on the way in. A shot cannot damage the core. The first force can make the trip in calm weather while it is within the fog line; the later artillery keeps the formation repulsed through swells. The route is a straight line for now.

The scout and introduction party are the only normal visits scheduled independently of cities. After the introduction, the city clock schedules arrivals only; raids are produced by factories. Rivals' small arms can answer colony fire at their vehicles, but the city and its buildings do not automatically shoot back. Rival structures are gray and subdued to distinguish them from the colony at every zoom. Wrecks still drop their own salvage and stolen oil as a pile for the colony's robots to haul.

The colony's guard post remains its short-range oil-paid answer; squads
remain direct orders through the war factory and keys 1-9. The player's
artillery remains an unlocked blueprint that shells visible rival targets
for lilac and oil. A war factory can also build a vulnerable mechanic,
which repairs damaged colony buildings with oil from its own tank. Bullets
and shells stay in the serialized state with hit, damage, wreck and
visual-effect rules. City structures can be selected and attacked by
guard posts, squads and artillery like other rival targets.

Open: independent city choices and production strategies, pathfinding, cities sending more than one sortie at once, the city's response to a completely guarded oil supply, and city graphics beyond gray versions of the existing silhouettes.

## Architecture
The game is a deterministic simulation first, and a picture of it second. These rules are law; every feature bends around them.

1. **One serializable state.** The whole game is a single value (`State`) that serializes to JSON with no pointers, no channels, no functions. `State.Version` identifies its save schema; `enterRegion` migrates old protector records and robot roles. Entities live in ID-keyed tables (`map[int64]Entity`-style, with fixed field structs); every reference between things is an ID, like a relational database. Saving = the state's JSON into the local database (SQLite, the schema a server keeps; `golib shot --save` feeds the same value through the shot channel). Loading the state = loading the game, exactly.
2. **Actions in, state out (flux/redux).** Nothing mutates the state
   except reducers. Actions are serializable structs (`MarkBuilding`,
   `QueueMechanic`, `OrderSquad`, `Tick`...). `Apply(state, action) ->
   state` is deterministic: the same action on the same state gives the
   same result. `Tick` is the action the loop sends 60 times per second.
3. **The game is a visualization.** `Update` reads input and produces actions; `Draw` renders the current state and changes nothing. All rules live in the simulation files, free of `golib.Input` and `golib.Screen`, so tests drive them directly.
4. **Determinism.** The simulation never reads the clock (dt is 1/60), never reads globals, and iterates entities in sorted ID order. Gameplay randomness comes from the state's own PRNG (a seed plus counter inside `State`), so a save reproduces its future; `golib.RandomInt`/`RandomFloat` are for looks only (cosmetic particles, menu clouds). A seed plus an action log replays any game - which is also the future multiplayer server: one authoritative sim, clients send actions, receive states.

The file layout follows that law: `state.go` holds the serializable
entities, including workers, troopers, mechanics and their production;
`actions.go` and `Apply` are the simulation's entry point; and `sim_*.go`
files, with their views beside them. Deposits remain terrain-shaped (one
remaining amount keyed by each deposit's heart tile), and the ground is
static data generated from `State.Seed`, never state. `play.go` and
`draw.go` only read and display the state. The technical map is in
[README.md](./README.md).

## Screens
- **Menu:** the game's name over the fog, the player's number under it (`player #30E99076` - the machine's identity, hashed), Play and Quit. Play carries the player to their base as they left it, or deals a new region when there is none. Esc quits; it is the only screen where it does.
- **Play:** the isometric region, the camera panning and zooming, the
  bubble drawn over the ground, robots shuttling, the fog line visible
  and creeping, and resource counters on top.
  - A click inspects a **cell**, the footprint of a building and the unit
    the player picks and counts by. The panel lists what stands there,
    one expandable card per thing: the building, site or pile on that
    cell and the robots crossing it, never the charger beside a silo.
    The outline is the selected cell's. A deposit's card and its pump
    show from any cell of the whole patch; the core's card shows from
    any cell of its pad. Cards start open by themselves: the cell's
    primary thing (deposits, buildings, the core) and any thing alone
    on its cell.
  - **A store shows how full it is** on its body, in a vertical bar
    filling from the bottom: a silo and charger in oil, a warehouse in
    lilac against all its roofs, and the core in both oil and lilac.
  - An expanded oil or lilac card offers `send robot` while another
    worker can be assigned. It lists assigned units as up to eight
    clickable portraits per page. A portrait opens that unit's card,
    wherever it is in the region; `recall robot` clears only its post,
    and `back to deposit` returns to the patch card. A robot standing
    on a cell also has its own card with what it is doing.
  - A **robot roster button** under the squad boxes opens a panel on the
    right. Reopen it or press its X to close. Boxed groups show builders,
    unassigned workers, workers by deposit and mechanics; each row shows
    the model, ID and current task. Selecting a builder or worker offers
    `assign deposit` and, when posted, `recall`. Assignment arms the
    pointer: click an oil pool or lilac vein to send that exact robot,
    or right-click to cancel. This includes assigning a builder to a
    deposit when the colony is in trouble; it carries only a third of a
    worker's load.
  - A building's or site's card has a trash can button. Demolished
    buildings leave their cost and overflow on their cell as a pile
    with a card of its own (see [Demolition and loose items](#demolition-and-loose-items)).
  - While rivals are in the region, the HUD says in red what they are
    doing and where (`raiders camped east, moving in 2:40`, `rival base
    west, level 2`). A news plate under the HUD, for `reportShowTicks`
    15 s (one minute for a city founding), reports their theft, camp, raid,
    return, new base, razed
    building or fallen base. Rival vehicles and bases have cards too.
    A war factory's card builds troopers, offers one mechanic and gives
    its squad orders; damaged things show their remaining health and say
    mechanics can repair them; picked guard posts and artillery show
    their reach; bullets and shells fly with their light.
  - Esc saves and returns to the menu.
The current rival HUD reports an approaching city crawler, construction,
an assembling battalion or a force on the move. City-building reports
replace the earlier passive-base/gun-up notices in the historical screen
description above.

The oil patch remains one selectable resource across its cells. Its pump
and any pump site appear only on the pump's cell; clicking the visible
pump body selects that cell, whose panel shows only the pump.

An oil or mineral card lists every robot assigned to the whole patch,
eight portraits per page. Clicking one opens that robot's card at the
deposit panel, even while it is away hauling or refueling. Its card can
recall just that worker, or return to the deposit; a dry patch releases
all its workers.

When the latest rival report is still on its plate but its location is
outside the view, a small red arrow at the screen edge points toward it.
The city-founding report and arrow disappear after one minute.
Unopened schematics do the same for the core, in the drop's color, until
the badge is opened. Both guides disappear as soon as their target is on
screen; they never move the camera or take input.

- **Pause:** the frozen region under a message. The fog does not advance while paused.
- **Colony lost** (later): when the core is somehow unreachable, or the player quits the region.

## Controls
| Input | Action |
| --- | --- |
| WASD or arrows | Pan the camera |
| Mouse wheel | Zoom toward the cursor, gliding between whole stops, each twice the last (pixel art stays square at rest) |
| Mouse right, held | Drag the view: grab the ground and move it |
| Mouse left, on a free cell of ground | Open the build menu on that cell; picking a blueprint builds it there. Until the first schematics arrive, the click inspects the cell instead |
| Mouse left, on the schematics badge over the core | Open the drop: the badge goes and the callout says what arrived; clicking the callout closes it, while a click elsewhere closes it and acts on the region |
| Mouse left | Select / inspect the cell under the pointer: expand a card, press its buttons, send another robot from a deposit, or click a unit portrait to open its card and recall it individually |
| Mouse left, on the robot roster button | Open or close the colony robot panel; its X closes it too |
| Mouse left, on a robot in the roster | Select it; `assign deposit` arms an individual order, then click an oil pool or lilac vein to assign that robot |
| Mouse right, after `assign deposit` | Cancel the individual assignment |
| Mouse left, after a pump's `lay pipe` | On the ground: a bend of the pipe. On a ringed tank (silo, charger, core): the pipe's end, which marks it. On the pipe's last node: its menu (`connect` to the nearest store, `undo`, `cancel`). Right click: the menu away, the last bend back, or out of the mode |
| Mouse left, after a war factory's `give order` | On a rival vehicle: the squad attacks its party, that vehicle first. On the ground: the squad guards that spot. Right click: the order away |
| 1-9 | Call a squad: 1 is the first war factory raised, 2 the next. The key arms the order the same way `give order` does (one click orders, right click puts it away); the same key again takes it back. A squad's box at the top right - tank icon, unit count, the key below - calls it too |
| Mouse left, on a squad's pennant or ring | Call that squad, where it stands |
| Mouse left, on a card's trash can | First press arms it (`demolish?`), the second demolishes; a click anywhere else disarms |
| Mouse right, clicked | Close the menu / deselect |
| Esc | In the region: save and return to the menu. In the menu: quit |
| F11 or Alt+Enter | Fullscreen on and off |
| Control held, two clicks on the game's name | The dev tools on and off (see [Dev tools](#dev-tools)) |

## Dev tools
For whoever works on the game, not for the player: in the region, hold Control and click the game's name twice (within `devDoubleClickTicks` 24 updates) and a strip of buttons opens under the HUD, in two rows, `dev` beside the name; the same gesture closes it, and it stays open across a trip to the menu. The strip is view (`dev.go`), and what its buttons do are actions like any other, so a log with them in it still replays:
- **hold a swell / let the swell go** (`DevHoldSwell`): raises a swell at once and holds it up - `Fog.Held`, under which a swell doesn't drain - until the button lets it go, which ends it. A held swell is not one of the fog's own: `Fog.Swells` doesn't count it, so the swells to come are no sooner, longer or deeper for it. While it is held the calm's countdown waits.
- **reset world** and **replay seed** (`DevResetWorld`): reset world deals
  the region again from nothing on a random seed different from the current
  one; replay seed uses the current seed to deal the same map again. The seed
  shows beside `dev`. Both save at once, so the old base is gone from the
  database too. What the scene held of the old region (the picked cell, an
  open menu, a pipe in hand, the mites) goes with it. It is the way out of a
  save whose ground the generator has since changed.
- **rivals: next visit** (`DevNextVisit`): brings the next timed arrival now, the scout/intro raid before the cities and a city crawler afterward. It waits while a non-settled party is moving.
- **rivals: stop waiting** (`DevHurryRivals`): ends the wait of a camped intro party or city battalion, so it moves on the next tick.
- **rivals: new city** (`DevNewCity`): establishes a city now, up to `cityLimit` 3.
- **finish city build** (`DevFinishCityBuilding`): completes one next building in the oldest city.
- **finish battalion** (`DevFinishCityBattalion`): creates the oldest city's next complete force, assembled and waiting.
- **send battalion** (`DevSendCityBattalion`): ends that city's current assembly wait; it starts moving on the next tick.
- **fast forward x8**, under `replay seed`: the play scene sends `devFastTicks` 8 ticks an update instead of one until the button is pressed again, so a wait passes sooner. It is view, not state: the ticks are the same ones, so a game played fast is the same game.
- **place robots** (`DevSpawnRobot`): arms the pointer, and every left click on the region puts a built robot there, its tank full, for nothing - in the fog too, which is what it is for. The button again, or a right click, disarms it. While armed, clicks don't inspect or open the build menu.

## Rules (MVP)
- The robot factory's schematic waits over the core from the start. The
  first delivered load brings infrastructure (silo, warehouse, charger);
  the guard post comes when the scout's theft is inevitable, the frontier
  kit (protector, pump, pipes) at 5:30, the war factory after the
  introductory raid, and artillery when a rival city completes its war
  factory. A drop's badge waits until opened; the menu offers only what
  the cell could really take, and unaffordable options stand washed out
  with their shortfall boxed in red (see [Construction model](#construction-model)).
- One generated region, a pure function of `State.Seed` (`worldgen.go`): a gentle relief with great plains, ground cover in zones, oil pools and lilac veins that thin out toward their rims, fog everywhere else. Buildings ask for a flat cell.
- The core gives the colony one white builder, with a full oil tank. It
  is no more immune to the fog than any other builder: empty outside a
  bubble, it is digested. The factory builds either another builder or a
  blue worker for the same lilac and oil cost and 12 seconds of work.
- Builders first bring home any load they already carry and refuel when
  low, then raise buildings and lay pipes before collecting piles or
  working their own deposit post. Workers carry loads home, refuel, finish
  loading, collect piles, work their assigned post and idle; they never
  build or lay pipes. Both kinds have the same speed and tank. A builder
  carries only one third of a worker's oil or lilac load, enough for an
  emergency, not routine hauling. Troopers follow their squad; mechanics
  repair the oldest damaged building and then wait at their war factory.
- `send robot` on a deposit card assigns one unassigned worker and never
  takes a worker from another post. The right-side roster shows builders,
  unassigned workers, workers grouped by deposit and mechanics, with each
  unit's current task. Select a builder or worker and assign a deposit to
  order that exact unit; this is the explicit way to retask a builder for
  emergency hauling. Deposit cards still show up to eight assigned-unit
  portraits per page. A dry deposit releases its posted units.
- Idle builders and workers return to the core and line up in ranks before
  its broad face, five to a rank; ranks close when one leaves, and past ten
  the rest stand inside the first ones with a count label. A post that runs
  dry releases its units. Every load ends at the nearest store of its kind
  (warehouse or core for lilac, silo or core for oil).
- Any building but the core can be demolished from its card, and a site cancelled. Its tasks die with it, and its whole cost, plus what the stores lose the roof for, falls on its cell as one pile of loose items that the robots haul back to the stores. A protector can't go while its current bubble is the only one over another building.
- Rivals come for the oil, one moving party at a time: a scout four minutes in, which siphons 25 L and leaves its mark, then an introductory raid from the scout's bearing. Thirty cycles after the raid leaves, a crawler arrives and establishes a city that builds a repulsor, extractors and a war factory. The factory sends raiders first without artillery, then mobile artillery. Cities do not attack by themselves; their mobile forces do. Structures can be destroyed, wrecks drop loot, and the colony's artillery shells visible rival targets for lilac and oil (see [The rivals](#the-rivals)).
- A war factory builds troopers, up to six, and they are its squad: one
  click orders them to guard a spot or attack a rival party, a chosen
  vehicle first. It can also build one mechanic. The mechanic is not in
  the squad and takes no orders: it seeks damaged buildings automatically.
  Troopers do no work and shoot from their own tanks. Rivals shoot
  troopers, mechanics and guard posts; a post they bring down falls into
  a pile like any building.
- Lose nothing at the start: the core is indestructible and its bubble has no upkeep. Protector reserves are not part of the oil rivals can steal. The rivals take oil, never the game.
- Lose units: any builder, worker or mechanic outside a bubble with
  an empty tank is digested by the fog. Rival bullets and artillery
  blasts can kill troopers and mechanics; ordinary workers are not
  targets. A lost unit leaves 25% of its build cost and remaining tank
  in a pile, plus 25% of any cargo.
- Lose buildings: rival mobile artillery brings buildings down into a
  pile of half their cost, the core excepted. Only a mechanic repairs
  damage: 6 points a second, spending 0.2 L per point from its own tank.
  Mechanics have 60 hull; an isolated one cannot out-repair continuous
  artillery fire. They must be built, fueled and kept alive.
- The safe zone feeds you: an oil pool and a lilac vein, the region's smallest, sit whole inside the bubble, off the core, so at least one resource of each type is minable in comfort whatever the fog does outside.
- Guard posts, the war factory and artillery are the colony's arms, all three under a bubble like any building, all three paid in the stores the colony grows on: a post's shot in oil, a trooper in both and its shots out of its own tank, a shell in both.
- Buildings grow the colony out of the bubble: the oil-fed shadow protector holds a small bubble of its own, and only under an active bubble may any other building stand. Silos and warehouses give the stores a bigger roof; the charger refills built robots' tanks away from the core; the factory turns lilac and oil into new robots.
- Four unit roles: builders (white) construct and lay pipes, workers
  (blue) harvest and haul, war-factory troopers fight as squads, and
  mechanics repair buildings. The starting builder is a gift; every
  factory-built unit costs lilac and oil. Builders, workers and mechanics
  burn oil from their tanks **while carrying something, and at no other
  time**; troopers pay for their shots. Low, a tanked unit walks to the
  nearest charger or the core to refill; empty outside a bubble, it is
  digested, regardless of whether it is a builder or worker. Fog drags at
  every walker's pace, and the bubbles cancel it.
- The stores have a roof (the core's own room, plus every silo and warehouse); a robot hauling into a full store waits at the core with the cargo in its arms.
- Lose the region (later): when oil hits zero inside the bubble and no robot can reach fuel, the bubble collapses; that ends the run.
- Difficulty grows with the fog's swells: they press harder and come more often, but never take ground for good. The quiet pressure is depletion - the near patches drain first, so haul distances grow on their own.
- No win condition in the MVP; the region is the tutorial for the arc.

## Art
The screen is 1280x720 with `Config.PixelArt`, scaling by whole numbers on
a larger monitor. Three monitor filters run in order (`shaders/glow.fs`,
`shaders/crt.fs`, `shaders/soft.fs`): restrained glow, a faint tube screen
and a small blur that rounds pixel corners. F2 turns them all off.

The region uses a manual isometric projection (`screenX = (x-y)*unitW/2`,
`screenY = (x+y)*unitH/2`), drawn back to front. GoLib does not load Tiled
isometric maps, so the ground is generated and drawn here. Movement is
projected to the screen and rounded to eight directions, 45 degrees apart
*on the screen*. Facing stays put when a unit stops and older saves default
to right; those screen headings become nonuniform world yaws in the
Blender renders.

All eight moving chassis have editable Blender models, rendered into eight
transparent PNG frames apiece with the same 2:1 orthographic camera and
lighting. Each also has an eight-frame shadow mask rendered from its 3D
geometry on a flat ground plane, with soft daylight from the model's light
direction. The game paints that mask beneath its matching unit sprite; it is
view-only and never enters the simulation. The colony silhouettes stay
distinct: narrow ivory builders with warm rear lights, teal workers with a
raised tank and rails, and broad olive troopers with a turret and forward
gun. A mechanic has a service chassis, amber tool deck and raised repair
crane, with no weapon.
Rival scouts are small and red, raiders carry an oil drum, crawlers carry
a repulsor mast, and mobile artillery has its cannon raised above its
turret. Their wheels and shaded sides have real 3D geometry. Portraits
and squad icons show these same models, scaled for the UI rather than the
world. Cargo, refuel blink, damage and oil carried remain view overlays;
the simulation and saved facing do not change.

The single `studio.py` makes the geometry primitives, materials, light,
camera, world-yaw conversion, model sheet and shadow mask for all models.
GoLib draws each mask and its model at the same projected ground point in
screen pixels, scaled to the camera's zoom; rounding a sprite in world
pixels before zoom would make it jump by tens of screen pixels. Blender is
only for editing and rendering the art, never for building or playing the
game.

## Sounds
Mix adjustment (2026-09-25): the oil drops keep to the ground. Their
bloops - the drip and its thicker cousin - play at half volume, and a
zoom gate (`dropHushZoom` 28, `dropHushSpan` 4) stills them as soon as
the view rises off the closest stop, so zooming out no longer carries
them over the whole region. The mineral ring wears an amplitude
modulation of `ringCycle` 20 s (`ringSwell`): it swells up, falls silent
and swells again, a little while sounding every so often, instead of
droning on. The crystal ping is 6 dB up (`tinkVolume` `0.2125 / 24`),
still almost inaudible by design.

Mix adjustment (2026-09-24): the crystal ping is four times quieter and
has a six-pulse-per-second amplitude modulation; the continuous mineral
resonance is down from 0.28 to 0.05. Ordinary world sounds now use a
steeper 3.2 falloff and a range of 1.5 screen-diagonal radii, capped at
2800 m, so they are nearly silent beyond the view. Cannon reports keep
their 3800 m reach, 1.3 falloff and raised listener, so distant shots
remain audible.

Mix adjustment (2026-09-24): ordinary world sounds now drop with the
remaining reach raised to 2.3, rather than linearly; cannon reports use
1.3 and retain their longer 3800 m reach. Interface clicks play at 0.72
of their former pitch. Marking a building site (including a pump) makes
a low, soft thump instead of another click. Each new rival report sounds
a short, low three-note brass fanfare, made by `tools/soundgen` as
`assets/sounds/alert.wav`; reports already present in a loaded save are
silent. The fanfare is interface audio, independent of camera distance.

Audio correction (2026-09-23): the listener is the camera's ground center
in meters, lifted as the zoom wheel pulls out (up to 2100 m). Every world
emitter uses the same three-dimensional distance to that listener, fading
over 2800 m (3800 m for cannons); zooming out makes the world noticeably
quieter, while moving toward a turret never makes it quieter at the same
zoom. Artillery fires from the gun's position with enough reach to hear
it from the core; the landing sounds from its own position. The pure tone
was the shell's sine
whistle, not the mineral ring: it now follows each shell throughout the
descending half of its arc at a much lower volume, changes level as the
shell and camera move, and stops on impact. Crystal pings are half as loud
again.

Debt: gun reports and artillery recordings take their emitter-to-listener
volume when they start; GoLib's one-shot `PlayWith` offers no per-voice
volume control for a sound already playing. Their level does not follow a
moving camera until the next shot. The whistle and ambience loops do
update their level each tick. Per-voice spatial control belongs in the
framework's audio API if these reports need to track the moving view.

Mix adjustment (2026-09-23): the mineral ring now wanders at irregular
intervals through pitches spanning at most one semitone, generated as a
seamless eight-second loop by `tools/soundgen`. The pings rise another
octave (3840, 5120, 6400 Hz) and drop to one third of their previous
volume; their timing and the ring's level are unchanged.

Mix adjustment (2026-09-23): the oil bed reaches full volume up close
and falls off more gently toward the edge of the view. Crystal pings are
twice as frequent (0.2-1.25 s apart before the crowd multiplier), an
octave higher and a quarter as loud; the continuous mineral ring stays
as it was.

Audio revision (2026-09-23): world positions are projected into the camera's
pixel space before attenuation. A cannon, bullet, impact, pool or vein now
sounds where it appears, at every zoom; off-screen sounds fade beyond the
view. The wind sits lower in the mix, close-range oil has a stronger bed and
bubbles every 2-7 s (gurgles every 10-25 s), and the crystals carry a quiet
continuous, wavering ring beneath lower-pitched, clearer pings. The shell's
landing has a deeper noise burst. Audio observes each simulation tick, so
fast-forward cannot swallow a short-lived shot between updates.

Landed (2026-09-22), in `audio.go`, all of it view: the world speaks where it happens and the view weighs it. Every world sound is multiplied by how close the view stands (`nearness`: a whisper at stop 0, whole from stop 3, never nothing) and by its distance to the view's middle (a little past the view's width on screen), so far out the world whispers under the wind. The wind loop and the oil pools' buried seethe are synthesized by `tools/soundgen` and read as files (`wind-loop.ogg`, `oil-bed.ogg`), so they loop with no seam; the wind is three layers driven by one long gust - a deep rumble always there, an air that swells with it, a whistle only the strongest gusts sing - so being far out sounds like the atmosphere and not like a fault. the bed lives at the nearest pool with oil left (never a dry one) and drops a bloop (`oil-drip.ogg`) every 5-20 s and a thicker gurgle (`oil-gurgle.ogg`, CC-BY) every 30-70 s, while the lilac veins, the minerals, sparkle: a soft crystal ping, one of three pitches varied by the play, every 0.4-2.5 s at the nearest vein with ore - and the more veins the view hears, the louder and the sooner the next ping, so the shimmer grows with the mineral in earshot. The war's shots are learned the way the lights learn them, by comparing the state's with the ones seen last: the colony's artillery its cannon recording (`artillery-fire.ogg`, CC0), a rival base's gun the filtered, echoing one of the same (`artillery-fire-distant.ogg`), small arms two short reports (`gun-a/b.ogg`, CC0) held to one sound every few ticks, a shell in the last second over the view falls whistling (a falling note made in code, once per shell), and its landing is a wide whump of noise, made in code too. The interface clicks (`click.ogg`, CC0): opening the build menu, picking a group or a blueprint, every card's button, the schematics' badge, calling a squad, the trash can's two presses. Still to come: the fog's own low loop outside bubbles, the repulsor hum, robot blips, a digestion crunch, a construction chime, sirens. Fully playable muted.

## Tuning
In `audio.go`: `audioFalloff` 3.2 and `audioViewReach` 1.5 for world
sounds, capped at `audioReach` 2800 m; cannon reports use `shellFalloff`
1.3 and `shellReach` 3800 m. `tinkVolume` is `0.2125 / 24`, and
`ringVolume` is 0.05 under the ring's `ringCycle` 20 s swell. The oil
drops are `dripVolume` 0.3 and `gurgleVolume` 0.275, and keep to the
closest stops: `dropHushZoom` 28 stills them and `dropHushSpan` 4 fades
them up to whole voice - provisional, to tune by ear. `uiClickPitch`
0.72, `placeVolume` 0.7 and `alertVolume` 0.65 set the interface mix.

Lost colony units leave `unitWreckRefund` 0.25 of each resource: build
cost, cargo and remaining tank. The wreck is a pile on their cell
(`sim_piles.go`). Fog-digested workers and fallen combat units use the
same rule. A mechanic costs `mechanicCostLilac` 100 kg and
`mechanicCostOil` 50 L, takes `mechanicBuildTicks` 900 ticks to build,
has `mechanicHealth` 60 health and repairs at `repairPerSecond` 6
damage/s for `repairOilPerPoint` 0.2 L per damage (`sim_squads.go`,
`sim_shots.go`).

An unprotected finished pump loses its full health over `pumpFogTicks`
600 ticks (10 s) in `sim_buildings.go`; `pumpMitesMin` 15 to
`pumpMitesMax` 100 in `mites.go` grows its visible swarm as it fails.

Pinned as code lands, all at the top of the sim files with units in the name: `fogCycleTicks`, `protectorOilPerSecond`, `robotChargeSeconds`, `robotMoveSpeed` (units/s), `oilPerPoolUnit`, `lilacPerVeinUnit`, blueprint costs. Pinned so far, in `region.go`: `regionCols`/`regionRows` (25x25 tiles), `tileW`/`tileH` (48x24 px at 2K/2), `unitsPerTile` 200 (the world's unit is a meter: a tile is 200 m across, the region 5 km; a robot is 6 u across, the core's monolith 16 u, a future building 40 u, a deposit patch of four tiles is 400 m aside), `coreBubbleRadius` 4 tiles, `fogLineRadius` 10.5 tiles, `fogFadeTiles` 2.4 tiles. In `play.go`: `zoomOut`/`zoomIn`, the wheel's stops, each twice the last: 1, 2, 4, 8, 16, 32 (stop 0 shows the whole region as icons, stop 5 about 80 m of ground; at 32 a robot's 6 u is about 46 screen px), `zoomGlide` 0.1 s (the zoom glides from stop to stop instead of jumping, keeping the point under the cursor under it; only at rest is the zoom a whole power of two), `panSpeed` 480 screen px/s, constant on the screen at every zoom; the right button drags the view, grab style. The camera is view, not state: it lives in the play scene and never serializes. In `draw.go`: `propZoom` 4, the zoom from which the rocks and bushes are drawn; they are world-sized (6 u across), so zooming in grows them from pebbles to boulders. Robots draw at their world size but never under about 3 screen px (`dotRadius`): far out, everything alive is a point, R.U.S.E.-style. The core draws as a dark monolith on its tile's middle (`drawCore`), broad face to the right, a seam of light down it and a top in the core's warm white, the part the glow filter picks; it obeys the buildings' icon law and sorts by depth among them, and its pad, the whole tile, lies on the ground under the robots. Deposits draw as one continuous body per patch - a pool as one sheet of oil, a vein as one shelf with crystal clusters over its tiles - shrinking as the patch drains, one big scar when dry. The fog's shape and speed are the feel of the game: their section is [The fog](#the-fog), and their dials are pinned, in `sim_fog.go`: `fogCycleTicks` 1800 (30 s a cycle), `fogSwellPeriod` 18 cycles (the first swell at 9 min), `fogSwellQuickener` 0.90 (each swell shortens the next calm; `fogSwellMinPeriod` 4 cycles), `fogSwellTicks` 900 (15 s) growing `fogSwellTicksGrowth` 180 apiece to `fogSwellTicksMax` 3600 (a minute), `fogSwellReach` 2.0 tiles growing `fogSwellGrowth` 0.35 apiece, capped by `fogSwellMargin` 0.75 tiles the line never takes off the bubble, `fogSwellSpeedFactor` 0.25 in the pushed band (fog that was already there keeps `fogSpeedFactor` 0.5) and `fogSwellBurn` 1.5x outside the bubbles.

In `sim_tech.go`: the factory blueprint is the opening drop, and the
first delivery unlocks infrastructure. `techFrontierTicks` 5:30 is the
frontier kit's clock drop. `legacyTechIndustryTicks` keeps the former
7:00 factory trigger for migrating saves without a tech ledger. The view's
`techBadgeR` is 22 px, `techPulseTicks` is 90 (one breath of the glow,
from the state's tick) and `techCalloutW` is 280 px.

In `sim_cities.go`, `cityAnnouncementTicks` is 3600 ticks (one minute).
The city status, founding report and offscreen arrow share that deadline;
other reports last `reportShowTicks` 900 ticks (15 s). In `guides.go`,
offscreen arrows sit `guideEdgeInset` 28 px from the
side edges, 78 px below the top and 52 px above the bottom, clear of the
HUD and the bottom help line.

Protector fuel, in `sim_buildings.go`: `protectorOilCap` 200 L,
`protectorCostOil` 40 L of initial charge, `protectorOilPerSecond` 0.25 L/s
and `protectorRadiusFadeBelow` 5%. These are provisional economy dials.
The radius fades linearly below the threshold and is zero with an empty tank.

Pump picking, in `inspect.go`: the visible isometric body selects the pump's
cell. The deposit remains one functional patch, while pump and site cards
stay local to that cell.

The region is generated from `State.Seed` (`worldgen.go`, its laws tested over 40 seeds in `worldgen_test.go`; `defaultSeed` 0 is a new game's and an old save's). **Relief** comes from wave function collapse over landform blocks (`reliefBlock` 4 cells, 100 m): each block is a wave over `reliefLevels` 4 levels (0 basin, 1 plain, 2 and 3 hills), neighbors - diagonal too - differ by a level at the most, the blocks within `reliefCorePlain` 2.2 tiles of the core are pinned to the plain and `reliefMarks` 16 hilltops and basins are pinned out in the region; then the block with the least left to decide collapses to a level drawn by `reliefWeight` times `reliefAffinity` for each neighbor already at it (the plain's pull, 6, is what leaves the great flats), and what that rules out spreads. The constraint keeps every wave an unbroken run of levels, so the collapse never contradicts. The levels land on the cells' corners after a smooth wander (`reliefWarpCells` 2.6) that takes the blocks' straight edges away; a level is `levelHeight` 4 m, a slope is one level to the cell (16%), a cell with four equal corners is flat, and `canPlace` asks for one: 95% of the ground is. The relief is looks and building ground for now - robots walk it at their one speed. **Cover** is three octaves of noise in zones `zoneWave` 28 cells wide, thicker in the basins, thinner on the slopes. **Deposits** follow `depositPlans`: one pool and one vein whole inside the bubble (hearts 1.7 to 2.4 tiles out, radius 6.5 to 7.5 cells) and two more of each with their hearts 7.2 to 8.4 tiles out (radius 8 to 12 cells), hearts `depositApart` 3.2 tiles apart and on flat ground, where the pump stands and the robots load. A body is a stretched, turned blob (veins 1.55, pools 1.15) whose edge a noise bends, mottled all over and broken into specks toward the rim; each cell has a richness from 0 to 1, and what a deposit holds is its richness times `oilPerRichCell` 70 L or `lilacPerRichCell` 240 kg - about 3 kL and 10 t by the core, up to 10 kL and 30 t far out. A tile belongs to a deposit when it holds `depositTileOre` 0.8 of richness and touches the heart's tile through others that do; the ore on the tiles that don't is dropped, and no two deposits share a tile.

The world speaks SI: `unitMeters` 1 (one world unit is one meter, in
`things.go`, so a tile is 200 m across, 4 ha); the core is a monolith in
the old proportions, 1 by 4 by 9 - `coreSlabDeep` 4 m, `coreSlabWide`
16 m, `coreHeight` 36 m - and its card headlines its 800 m bubble radius.
Each thing type gets its color from `catalog.go`, which falls back to a
stable hash of the type name; `markup.go` colors text with `[name]...[/]`.
In `inspect.go`, `tooltipWidth` is 290 px, `titleSize`/`textSize` are
15/13, and `buttonWidth`/`buttonRow` are 112/22. Deposit portraits use a
four-column grid, eight per page, and the panel stays anchored to the
selected cell's projected middle. `robots_panel.go` adds the fixed roster
under the squad strip: five entries per page, grouped by role and post,
with individual assignment and recall controls.

The robots, in `sim_robots.go`: `startingBuilders` 1, `robotSpeed` 30 u/s,
`robotCarryOil` 30 L and `robotCarryLilac` 20 kg per worker trip; a builder
carries `builderCarryPart` one third as much. `robotLoadTicks` is 150 (2.5 s
loading at a deposit). Builders alone raise sites and lay pipes; both roles
can collect piles and work an explicitly assigned post. The idle ranks by
the core use `parkSlots` 10 places, `parkRankSize` 5 to a rank,
`parkSpacing` 7 u and `parkFromCore` 10 u before the monolith's broad face.
A drained patch leaves one big scar and releases its assigned units. The
stores live in the state (`State.Stock`) with the room each roof gives them,
and show in the HUD and in the core's card; robot captions (`things.go`)
read the current task in simulation priority order.

The buildings, in `sim_buildings.go`, all landed: the ground's last subdivision is the **cell**, `buildingCell` 25 u on a side (8 by 8 to a tile, `regionCellCols`/`Rows` 200), the footprint of the smallest building - about four robots across - so buildings sit on cells and a tile may hold several. The build menu opens on the clicked cell over two rings - the build groups, then a group's blueprints - and picking a blueprint marks it there (dimmed when `canPlace` or the stores refuse it), paying lilac up front (factory 200 kg; charger 120 kg + 40 L; silo 100 kg; warehouse 100 kg; protector 180 kg + 40 L initial fuel) and asking `buildingWorkTicks` 600 (10 s) of robot work to raise; a building draws as an isometric body that never shrinks under a 9x6 screen-pixel icon, and a site under construction shows its built part rising from the ground in solid colors inside the wireframe of the whole body, the work's progress bar under the cell, all drawn over the fog, the builders standing on their cell's edge. The factory turns `robotCostLilac` 40 kg + `robotCostOil` 30 L into a robot every `factoryRobotTicks` 720 (12 s), one at a time, its card carrying the build robot button while idle. The stores' roof: `coreOilCap` 1000 L and `coreLilacCap` 2500 kg, `siloOilCap` +1000 L and `warehouseLilacCap` +4000 kg apiece, and they start with the core's gift (300 L, 600 kg). A built robot's tank: `robotTankLiters` 120 L, `robotBurnPerSecond` 0.25 L/s while it carries a load and nothing otherwise, `robotLowTankAt` 25% the line that sends it to the nearest refill post (a charger or the core, `chargerRefillPerSec` 20 L/s from the stores, filling to the top so it doesn't dance between post and charger); at zero outside a bubble the fog digests it. A shadow protector has a dedicated `protectorOilCap` 200 L tank; it starts with its 40 L construction charge and uses `protectorOilPerSecond` 0.25 L/s (provisional). Its `protectorBubbleTiles` 2.0 radius (400 m) stays full above `protectorRadiusFadeBelow` 5% charge, then fades linearly to zero. Robots and pipes refill it; its oil is reserved and can't pay other costs or be stolen. Old saves give existing protectors their starting charge once (`State.Version`). Only an active protector shelters buildings and robots; only a protector may stand outside every bubble - the fog's law in `canPlace`. The fog's drag, `fogSpeedFactor` 0.5, slows every robot, core or built, scaled by how deep its tile sits in the mist, and the bubbles cancel it. The robots carry no plan - every tick the rules derive what one does from the state (`stepRobot`), so a save reproduces its future; their positions are floats in units, the tile computed from them; deposits are patches (`Deposit` in `region.go`, flooded once out of the layout): one robot per patch, one card per patch, and what remains lives in `State.Drain` under the patch's key.

The colony saves itself, in `store.go` and `identity.go`: the machine's own ID (Windows' MachineGuid, macOS' IOPlatformUUID, Linux' `/etc/machine-id`), hashed with `playerIDSalt`, is the player identity - `playerIDSalt` "niebla player id v1", 64 hex characters, shown on the menu as its first eight (`#30E99076`) and never in raw form; a machine with no ID gets a random one kept in the database. The local database is SQLite (`modernc.org/sqlite`, pure Go - no C compiler here), at the player's settings folder in `GoLib games/niebla/niebla.db`, with the schema a server keeps: `players` (identity, source, created_at, `token` empty until a server hands one out), `saves` (the whole State as one JSON value per player and slot, `region` for now, with tick and timestamp) and `machine` (key-value for this machine alone, the fallback identity lives there). State version 2 migrates old `core` robots into fueled builders and old `built` robots into workers, including an in-progress factory product. Autosave every `autosaveTicks` 900 (15 s of game time) and on leaving the region, so a window closed without ceremony loses less than 15 s. Under `golib shot` and `go test` the database is `:memory:`, so shots and tests never touch the player's base, and `golib shot --save` still starts a game deep in a state: `resumeState` takes the seeded value over the database. The driver doesn't build for `js/wasm`: a web build will take its store from a server or the browser's own.

Demolition and loose items, in `sim_piles.go`: `demolishRefund` 1.0 (the part of the cost that falls to the ground; a site gives back the same), no work ticks to demolish, piles loaded with the deposits' `robotLoadTicks` and carry sizes, one kind per trip, lilac first. A robot never loads what the stores have no free room for, counting what is already on its way home (`freeRoom`), and unloads `storeStandoff` 11 u from its store's middle, spread by ID like the builders. State holds `Piles` (by ID: cell, oil, lilac) and `Robot.Pile` (the pile a loading robot stands at), the actions are `Demolish` (a building's ID) and `CancelJob` (a site's cell), `canPlace` refuses a cell with a pile, and the catalog has the `site` and `pile` types, primary on their tile. The robot's day is a list now (`robotDay`), the shape the per-robot task list will filter. In `inspect.go`: the trash can, `trashWidth` by `trashHeight` 11 by 13 px at the end of a card's title, red and under `demolish?` while armed.

The rivals, in `sim_enemies.go`: every dial is in [The rivals](#the-rivals). The state gained `Enemies`, `Parties`, `Raids` (the visits that were, the tick of the next), `Marks`, `Reports` (the last `reportsKept` 12) and `Rolls`, the counter of the state's own PRNG (`State.roll`, splitmix64 over the seed and the counter), which the rivals are the first to draw from: a visit's bearing. A party moves as one at the pace of its slowest, the members `formationOffset` around its leader (18 m and up, inside the crawler's pocket), and stops `siphonReachUnits` 40 m from its tank. `Building` gained `Reload` and `Aim` for the guard posts; a shot shows `guardFlashTicks` 6. In `enemies.go`: the news stay `reportShowTicks` 900 (15 s) on a plate under the HUD, the mark is `markScale` 1.3 (about 40 m long), vehicles never draw under 5 px across (the crawler 9), and bearings are named as the screen shows them, north up (`compassWord`).

The fog on the screen, in `mist.go`: `mistHaze` 0.3 outside the circles, `mistLayers` 16 steps from the haze to whole fog across `fogFadeTiles`, each layer the region minus the clear circles, drawn in strips `mistStepPx` 3 screen px across whose gaps join into quads, a run of untouched strips going out as one; the front is round now, where it used to step tile by tile. The ground is drawn out to `groundReach` (where the fog is whole, and a tile's corner more) and under any protector farther out (`groundShows`).

The squads, in `sim_squads.go`: every dial is in [The rivals](#the-rivals). A trooper is a `Robot` of kind `combat` with `Squad` (its war factory's ID), `Health`, `Reload` and `Aim`; its day has one line of its own, `squad`, after the tank's, and the working lines refuse it (`pickRobot` too). The orders live in `State.Squads` under the war factory's ID (`Squad`: order, spot, party, focus), set by `OrderSquad`; a squad with no entry guards its door (`squadOf`). `QueueRobot` builds for both factories (`robotWorks`). `Enemy` gained `Reload` and `Aim` for its gun. In `squads.go`: `orderPickPx` 16, the screen pixels around a vehicle that still pick it for an attack.

Shots, in `sim_shots.go` and `shots.go`: every dial of the first is in [The rivals](#the-rivals). The state gained `Shots`, `Building.Damage` and `Raids.Settle`; `Party` gained `Settles`, `Level` and `Grow`. The view's field: `fxGravity` 320 m/s2 on a spark, `fxMaxSparks` 1600, a pool of light stacked from `fxLightRings` 14 ellipses (a spark's own from 3), a shell's burst of 90 sparks, 14 puffs of smoke and a flash 190 m wide that lasts 0.45 s, a bullet's of 5 sparks.

Shells start `shellMuzzleOffsetUnits` 24 m ahead of their firing unit.
Their orange trail grows to `shellTrailLengthUnits` 54 m; `shots.go` adds
faint smoke every `shellSmokeStepUnits` 30 m, fading as it rises.

Current factory production differs from the earlier implementation note
above: its card offers `build builder` and `build worker`, at 40 kg of lilac
and 30 L of oil apiece. The role rules and opening gift are in
[Rules (MVP)](#rules-mvp).

## Prototype scope
The prototype is done when these five have landed, on top of the debts under [Later](#later) (decided 2026-09-21). Each is a heading to design, not a design, until its turn comes: they are discussed and landed one at a time. Enemies and battles is under way and nearly whole; the other four are not started.

The introduction note below records the original ladder from 2026-09-22.
Its unlock order was revised on 2026-09-25: the factory is the opening
gift, and the first delivery brings infrastructure. The live rules are in
[Rules (MVP)](#rules-mvp).

- **Enemies and battles.** The swell is pressure, but it is only weather: resources should also buy war. **Under way**: the design and what has landed are in [The rivals](#the-rivals). Decided (2026-09-21): no walls, ever, and war that is good to watch, R.U.S.E.-style (artillery, an enemy that builds a base in the region); a repulsor repels the fog and nothing else; rivals steal, and later destroy; they ride vehicles under a crawler's mobile repulsor, which is never usable loot; wrecks drop loot; many kinds in time, humans for now; positioning is coarse - posts (preventive) and small squads, never units placed by hand one by one; an enemy that settles is a warning with a clock, and a head-on attack is at a disadvantage. Landed: the scout and its mark, the camped raids, the fog's due, loot, the guard post, the war factory and its squad, rivals that shoot back at troopers, the settled enemy and its gun, buildings that fall and are mended, the colony's artillery, bullets and shells with their light and their bursts.
- **Economy analysis and balance.** Times, yields, costs and rates looked at as one system, so that the decisions are interesting instead of obvious: how long a deposit lasts, what a robot pays back and when, what a pipe saves against the legs it replaces, what a swell costs. `economy_test.go` is the first instrument: it plays the real, deterministic simulation for an hour over three seeds and records each minute in CSV. It compares safe harvesting, worker growth and a protected oil outpost with a pump, pipe and guard; it does not duplicate the rules in a spreadsheet. The first report (2026-09-22) says two core robots bring about 60 L and 40 kg a minute from the safe patches; the core oil tank fills around minute 13 and its lilac store around minute 44. The outpost opening reaches its protector around minute 2, pump around minute 5, guard around minute 11 and laid pipe around minute 17, so those are measured hypotheses, not tuning targets yet. Next reports change one dial or opening policy at a time, and compare the times to each milestone, reserves, resources mined, losses and whether the player had a usable answer before the threat. What remains deliberately unbalanced (2026-09-22): every dial of [The rivals](#the-rivals) was set by eye, a level 2 base razes a small outpost in a few minutes, and oil got much cheaper the day the tank stopped burning while a robot walks empty-handed. Reference: [Difficulty curves](https://www.davetech.co.uk/difficultycurves) (Dave Tech) - the **difficulty saw**: difficulty is not one rising line but a tooth per mechanic, a spike when it is introduced and a slope down as it is mastered, and later mechanics call back to earlier ones as foundations. It bears on the economy (the swells' growth is the base line the teeth ride on) and on the introduction below (the order of the unlocks is the order of the teeth).
- **Economy analysis and balance.** Times, yields, costs and rates looked at as one system, so that the decisions are interesting instead of obvious: how long a deposit lasts, what a robot pays back and when, what a pipe saves against the legs it replaces, what a swell costs. `economy_test.go` is the first instrument: it plays the real, deterministic simulation for an hour over three seeds and records each minute in CSV, including dedicated protector oil separately from spendable oil. It compares safe harvesting, worker growth and a protected oil outpost with a pump, protector-fed pipes and a guard; it does not duplicate the rules in a spreadsheet. The first report (2026-09-22) predates protector upkeep: it says two core robots bring about 60 L and 40 kg a minute from the safe patches; the core oil tank fills around minute 13 and its lilac store around minute 44. The outpost opening reaches its protector around minute 2, pump around minute 5, guard around minute 11 and laid pipe around minute 17, so those are historical measurements, not current tuning targets. Next reports change one dial or opening policy at a time, and compare the times to each milestone, reserves, resources mined, losses and whether the player had a usable answer before the threat. What remains deliberately unbalanced (2026-09-22): every dial of [The rivals](#the-rivals) was set by eye, a level 2 base razes a small outpost in a few minutes, and oil got much cheaper the day the tank stopped burning while a robot walks empty-handed. Reference: [Difficulty curves](https://www.davetech.co.uk/difficultycurves) (Dave Tech) - the **difficulty saw**: difficulty is not one rising line but a tooth per mechanic, a spike when it is introduced and a slope down as it is mastered, and later mechanics call back to earlier ones as foundations. It bears on the economy (the swells' growth is the base line the teeth ride on) and on the introduction below (the order of the unlocks is the order of the teeth).
- **Introduction to the game.** The player gets the elements little by little, so there is always one more thing they can do and never ten at once: the tutorial is the unlocking. The colony's buildings arrive as **remote schematics** the core receives: a drop of the ladder (`sim_tech.go`) lights a pulsing badge over the monolith, its click opens a callout that says what came in and how it is used, and the blueprints join the build menu - before the first drop the menu doesn't open at all, so the callout's "click empty ground" is true the day it is said. The first ladder, landed 2026-09-22 and tuned by the first play the same day: the first delivery of a haul home brings the **infrastructure** in (silo, warehouse, charger) - farming alone until then, no dead minutes on a clock; the guard post comes when the scout's drawing has become **inevitable** - a rival drinking at the tanks (`Enemy.Oil` over zero), or the mark already on the ground - and it comes **alone**, too late to stop the drawing, in time for the next visit; at 5:30 the **frontier kit** (protector, pump, and with them the pipes); at 7:00 the **robot factory**; after the first raid leaves, the **war factory**; when a base settles, **artillery**. The order is the difficulty saw (Dave Tech, above): a drop per valley, none in the middle of a peak, a war tool only once its lesson is on the road (the guard while the scout steals, the squads after the first raid, the artillery after the base digs in), three blueprints to a drop at the most. Two teeth still run on the clock on purpose - they hold the long calm between the scout and the first raid; the rest answer events the player has seen or caused. Candidate for the measured loop: the storage drops answering the roofs ("at 80% of an oil roof, the silo") instead of the clock. Plain milestone unlocks win for the prototype; a research center or choices between technologies wait until the measured loop says they need a resource sink or a strategic fork. A factory's extra workers still cannot raise the safe harvest while a deposit takes one robot - the multi-worker design under [Later](#later) comes first; until then the factory's callout promises hands, never ore. And nothing unavailable is ever shown: the menu offers only what the colony could raise right now (see [Construction model](#construction-model)), so the game gives itself away option by option and never spoils what it hasn't given yet.
- **Humans.** No walkers on the ground - the technology could draw them, the design doesn't want them - but buildings of theirs: housing, and whatever follows. To study whether to do it at all, because the next steps are food, waste and the rest, and the question is whether this game wants to be the next Ixion. What is decided is smaller: the colony gets **at least one more resource**, and people are one candidate for what needs it, not the only one. They are also the voices the mites' nickname already owes the game.
- **Electricity and connection.** Generators - oil-burning, wind, geothermal and the like - and a grid the buildings hang from. And **wireless points** that let robots coordinate: inside the grid's coverage a robot knows what the others are doing; outside it, it knows only what it is doing itself and where the stores are, so two robots out of coverage may walk to the same job or the same pile. The risk is teaching it, since it has little precedent as a rule about *knowledge*; the picture has plenty (StarCraft's pylon fields, Factorio's roboport ranges, Creeper World's network), so coverage drawn on the ground while placing, and a mark on a robot that has lost the grid, are where to start.

## Later
- **Debt - turrets must trade, not hold forever** (2026-09-23): guard
  posts already have 200 health, take rival bullets, show their damage
  and fall into a wreck; the rivals target troopers, mechanics and guard
  posts, but not artillery with their small arms (artillery can still
  take shell damage like any building). Let nearby rivals target an
  artillery piece too. Balance and test an unassisted guard post against
  a raid so it typically destroys about two vehicles before going down,
  with positioning, repairs and numbers still able to change the result.
  A post should buy time, not make the next raid safe to ignore.
- **Debt - the relief doesn't slow anybody** (2026-09-21): the generated ground is looks and building ground only, and robots walk it at their one speed. A robot should go slower uphill (and perhaps no faster downhill), by the slope under it along its way: `Region.heightAt` gives the height at both ends of a step, and `walkTowards` in `sim_robots.go` is where the factor goes, beside the fog's drag. It moves hauls' timings, so the tests that count ticks on seed 0 will need a look.
- **The arc (mid-game):** stabilizing the first region summons the ark, the mobile base - the game's own idea, arriving as a reward. By then the fog deepens too slowly to feel in an hour of play: a region is a chapter, not a home.
- **Expeditions:** sending a robot into the deep fog as adventure - richer finds, ruins, beacons, the fog's origin; the fear and the loot of the mist.
- **Server and multiplayer:** the architecture is already the protocol; a server runs `Apply`, clients send actions. Robots do the routine while players coordinate the colony.
- **Outposts network:** Subnautica-style forward bases, beacon relays that extend the robots' range; star pattern with the ark as hub.
- **Region map and travel**, logistics between outposts, the fog's origin as the endgame: reclaim the planet region by region.
- Sprite art over shapes; music (a tracker module or OGG when provided; `golib.NewTune` before that).
- **Text and translations:** all in-game text is English. Strings move to `assets/text/<lang>.json` (one flat key-to-string file per language, read once with `golib.ReadAsset`) when the first text-heavy screens land; the language is a player setting, not part of the simulation state.

## Changelog
- 2026-09-25: city status announcements and their offscreen arrow now last
  one minute from founding; later city reports keep the 15-second lifetime.
  The deadline is saved with the city, so loading an older city does not
  announce it again. Pinned by `cities_test.go` and `guides_test.go`.
- 2026-09-25: builders and workers are distinct unit roles. The colony
  starts with a vulnerable, fueled builder; the factory schematic is the
  opening drop, and the first delivered load brings infrastructure. Only
  builders build and lay pipes, while workers stay on their posts; builders
  carry a third as much and can be assigned to deposits in an emergency.
  A roster under the squad strip groups units by role and deposit, shows
  their current task, and assigns or recalls an individual by ID. Old saves
  migrate former core robots to builders and built robots to workers.
- 2026-09-25: pipe bands show each source's offered flow as orange length,
  with 90% orange at a pump's full 2 L/s; their phase follows liters moved,
  so a sink's limit slows them and a blocked pipe turns gray. Protectors
  fill before passing oil, then retain 0.25 L/s for upkeep and pass the
  surplus equally among their outlets. A chain of protectors narrows the
  bands by one upkeep per pylon. `Pipe.Offered`, `Pipe.Flow` and
  `Pipe.Moved` are saved for deterministic drawing and replay. Pinned by
  `pipes_test.go`, including filling before pass-through, the thinning
  protector chain, equal pump shares and flow-phase persistence.
- 2026-09-25: pumps can now be marked on outside oil pools without a
  protector. Once raised, an exposed pump is swarmed by mites and digested
  in ten seconds, leaving half its cost as a pile and a news plate that
  explains why; a protector over the pool stops the damage. The second
  pool's button, exposure, shelter and report have simulation tests, and
  shots check the swarm and the aftermath.
- Earlier opening ladder (superseded 2026-09-25): a delivered haul brought
  the factory, and its first worker brought infrastructure. This is kept
  here as history; the current order is described under Rules (MVP).
- 2026-09-24: the schematics callout shows what a drop brings as squares,
  in the likeness of the workers' portraits a deposit's card holds: the
  rendered icon of each construction above and its name below, one square
  per blueprint and, with the frontier kit, one for the pipes with a mark
  of their own (`techBrings`, `drawTechSquares`, `drawPipeIcon`). The dim
  list of names they replace is gone, and the dismissal hitbox grows by
  the squares' row so a click on the last one still closes the callout.
  Pinned by `tech_test.go` (what each drop squares, and the hitbox down to
  its last row); shots checked the infrastructure's three squares and the
  frontier kit's, beside the portraits they took their looks from.
- 2026-09-24: crystal pings are four times quieter and pulse at 6 Hz;
  the vein resonance is down to 0.05. Ordinary world sounds now fade
  against the view's on-screen radius, so sources beyond it are nearly
  silent; cannon reports retain their long, gentle falloff.
- 2026-09-24: shells now leave the artillery muzzle instead of the
  firing unit's center; their orange trail grows during the first 54 m of
  flight, and faint smoke puffs rise at 30 m intervals. A saved shot checks
  the muzzle, the growing trail and smoke at several frames.
- 2026-09-24: world audio fades harder with distance except for cannon
  reports; clicks are lower, marking a site thumps, and fresh rival
  reports announce themselves with a low brass fanfare.
- 2026-09-24: repairs now belong to a military mechanic, not the colony's
  ordinary workers. A war factory builds one for 100 kg of lilac and 50 L
  of oil in 15 s; the 60-health unit seeks the oldest damaged building,
  repairs 6 damage/s and spends 0.2 L per point from its own tank. It is
  outside the squad, cannot harvest or build, and can be killed by rival
  fire or the fog. Rivals can target mechanics as well as troopers and
  guard posts. An isolated mechanic loses ground to continuous artillery
  fire. Old saves with a war factory's untyped work still finish a
  trooper. Blender's `worker-mechanic` model and eight-view sheet give it
  a crane silhouette instead of a weapon. Pinned by repair, production,
  combat, wreck, refund, and old-save tests.
- 2026-09-24: deposits can employ several robots. Each send assigns a
  different worker, preferring an idle one and otherwise retasking the
  nearest worker from another patch; all workers share the deposit's one
  remaining amount and spread around its loading spot. Deposit cards show
  eight clickable portraits per page. A portrait opens that robot's card
  even while it is away; recall clears only its post, and the card returns
  to the deposit. Pinned by the shared-drain, individual-recall, portrait
  hit-testing and page-layout tests; shots checked both pages and a worker
  card opened from the second page. The saved state shape is unchanged.
- 2026-09-23: unaffordable blueprints came back to the build menu,
  washed out. The rings keep holding only what the schematics, the
  ground and the fog allow - hiding what the stores alone refused read
  as "the technology hasn't arrived yet", and the player went looking
  for a building that was never taken off the menu for money. Now a
  blueprint the stores can't pay stands washed out to gray, its group
  with it while nothing in the group could be paid; a click on it
  refuses with a dull click of its own and the menu stays; and the tip
  beside it names the blueprint, prices it and boxes in red each
  resource that falls short, with a `short of ...` line under the price
  (`radialOffered` no longer asks `canAfford`; `radialTipOf` builds the
  tip, `drawRadialTip` paints it). Pinned by `radial_test.go` (the
  rings keep their options over an empty store, the washed click
  refuses) and `TestTheRadialTipMarksWhatFallsShort`; shots checked the
  washed rings and the tip's red box.
- 2026-09-23: robots now raise marked protectors before other
  construction jobs, while keeping the order in which jobs of each kind
  were marked. Pinned by `TestRobotsBuildProtectorsBeforeOlderJobs`.
- 2026-09-23: the shadow protector moved from the military group to
  logistics: it expands and maintains safe ground rather than fighting.
  Pinned by `TestProtectorBelongsToTheLogisticsRing`.
- 2026-09-23: edge arrows now point toward the location named by the
  current rival report and toward the core while schematics wait offscreen.
  They disappear when their target enters view and are purely indicative.
- 2026-09-23: an open schematics callout no longer eats a click elsewhere
  in the region. Clicking the callout still dismisses it; clicking empty
  ground dismisses it and opens the build menu. Pinned by
  `TestDismissingTechCalloutPassesOutsideClicksThrough`.
- 2026-09-23: clicking a pump's visible body now selects its own cell.
  Pump and pump-site cards stay on that cell; the pool keeps its deposit
  card elsewhere, while still owning one pump for placement and extraction.
  Pinned by `TestAPumpAndItsSiteBelongOnlyToTheirCell`.
- 2026-09-23: colony protectors now have dedicated oil tanks, paid for
  with their blueprint, refilled by robots or pipes and excluded from
  general spending and rival raids. Their radius fades below 5% charge;
  empty protectors shelter nothing. Old saves migrate with one initial
  charge. Upkeep and capacity remain provisional; the economy probe now
  reports protector fuel separately and pipes pump oil through outposts.
  A shot checked the half-powered ring and the tank/radius details on its
  card.
- 2026-09-23: robots and mobile rival vehicles now turn to one of eight
  screen-facing directions as they move, then keep that facing while
  stopped. Their polygon hulls, nose lamps, rear cargo and combat details
  turn with them. Old saves default to facing right; orientation tests and
  screenshots check movement, persistence and the silhouettes.
- 2026-09-23: lost colony robots now leave 25% of their build cost and
  carried resources in a wreck, whether the fog digests them or enemy fire
  brings a trooper down. Pinned by the worker and trooper wreck tests.
- 2026-09-23: revised the city foundation: the arriving crawler remains
  as a construction rig, raises the antimist pylon first, then builds the
  Nexus. The pylon-first progression is covered by the city construction
  test and capture.
- 2026-09-23: moved new cities to 10 tiles, beyond the colony's
  artillery range. The arrival crawler is a construction rig; the city
  builds its antimist pylon first and its Nexus second. Oil-extractor
  droplets now rise smoothly and fade to transparent with a peak alpha
  of 0.34.
- 2026-09-23: replaced independent surprise raids and a passive base
  cannon with city-founded production. The scout and the introductory
  raiding party remain; they share the scout's saved entry bearing.
  Thirty cycles later a crawler arrives to establish a city, which builds
  a repulsor, oil extractor, lilac mine and military factory from finite
  local reserves. Its first sortie is a direct raider battalion without
  mobile artillery; the second brings artillery with its own antimist
  bubble. Added deterministic city-arrival/build/sortie tools, subdued
  gray structure art, city cards and reports, old settled-save migration,
  and tests for arrival, production, sortie composition and JSON replay.
- 2026-09-23: halved the crystal ping volume again; the mineral ring
  keeps its level.
- 2026-09-23: based positional sound on source-to-camera distance in
  meters, with camera altitude set by zoom; restored artillery's range,
  extended each shell's much quieter whistle across its descent, and
  halved the crystal pings again. Recorded per-voice playback limits as
  audio debt.
- 2026-09-23: replaced the ring's periodic vibrato with a seamless,
  randomly wandering pitch limited to one semitone, and made the crystal
  pings an octave higher and three times quieter.
- 2026-09-23: kept the mineral resonance, made its pings quieter,
  higher and more frequent, and brought the continuous oil bed forward.
- 2026-09-23: fixed world-audio attenuation using the camera's projected
  coordinates, restored gunfire and shell landings at their visible positions,
  lowered the wind, strengthened oil, and added a continuous mineral ring
  under clearer pings. Audio follows every fast-forward tick.
- 2026-09-22: being shelled finally sounds like being shelled. What the far view heard was the enemy base's gun firing - the one long boom that carries over the wind - while up close that firing rightly dies with the distance (the piece stands 1.5 km off) and the impacts were all there was: a burst that turned out to play at half its volume, `golib.Explosion` carrying a `Volume` 0.5 of its own. Now the landing is a wide whump made in code at full strength, and a shell in the last second of its flight over the view falls whistling - one falling note per shell, the first of the sounds the design promised for the war. The world's weight at the farthest stop went 0.25 to 0.15, so distant war is a murmur under the wind again. Pinned by `TestAShellWhistlesOverItsLastSecondOfFlight`.
- 2026-09-22: the rivals answer a post's fire, and the posts are spendable. `stepEnemyGuns` aimed at troopers and at nothing else; now it aims at the nearest trooper **or guard post** in reach, whichever stands closer (`nearestGuard`), and a bullet that flies at a building names it (`Shot.Building`) and hurts it on arrival - a post stands its 200 health, wears its damage bar, and at nothing falls into a pile of half its cost, the news saying so. Placing a post is now also spending it: it holds the ground while it stands and leaves it open when it falls, and the robots mend it between raids. Pinned by `TestARaiderShootsAGuardPostAndThePostCanFall`. The same listen turned the world's sound up - the zoom's weight never reached nothing, so at rest zoom the whole region was mute under the wind: `nearness` now floors at 0.25 and is whole from zoom 3 (was from 4), and the artillery, the small arms, the bursts and the minerals' pings all step forward a notch.
- 2026-09-22: the wind sits back and the minerals step forward, after a listen. The gust was well made but ever-present and too loud: far out it went 0.5 to 0.3 and up close 0.15 to 0.05, so it carries the far view instead of covering the ground one. The veins' pings were all but inaudible: base volume 0.3 to 0.5 and the notes themselves to full. And they now answer how much mineral there is, as they should have from the start: `nearestDeposit` counts the veins the view hears at all, and every extra one lifts a ping (up to 1.4x) and hurries the next along (up to twice as soon) - one vein twinkles alone, a district of veins crackles.
- 2026-09-22: the infantry can be heard, and the minerals sparkle. The small arms' reports were near-silent unless the battle stood on the view's very middle: their base volume went 0.4 to 0.55 and sounds now carry over the whole view's width, not three quarters of it. The veins, the minerals, got their own voice against the pools' seethe: a soft crystal ping (`golib.NewSound`, three Triangle-wave pitches of 2.1/2.75/3.4 kHz, no file), one every 0.4-2.5 s at the nearest vein with ore, its volume the vein's own audibility. `audio_test.go` pins the doors without ears: a new bullet within the view's reach spends the guns' cooldown, one out of earshot doesn't, and the ore's loops live only where the view stands.
- 2026-09-22: the far wind made wind. It was already the rule that the zoom weighs every world sound and far out only the wind is left - louder the farther out you stand, quiet when the ground is close, by design and not by bug - but what sounded was a flat bed of noise. Now one long gust drives three layers (`tools/soundgen`): the deep rumble that never leaves, the air that swells with the gust and a whistle only the strong gusts sing, calmer valleys between. Still `wind-loop.ogg`, still seamless.
- 2026-09-22: the region sounds (`audio.go`, `assets/sounds/`, `tools/soundgen`). The world speaks where it happens: a shell from the gun that fired it - the colony's artillery its cannon (CC0, Thimras), a rival base's the filtered, echoing one of the same -, small arms from the muzzle (two firework reports, CC0, rubberduck, held to one sound every few ticks so a battle doesn't rattle), and a shell that lands bursts in code (`golib.Explosion`). The zoom weighs every world sound - the closer the view stands, the louder the world - and the distance to the view's middle does the rest, so far out only the wind is left, a synthesized loop that swells as the view pulls out. The oil pools seethe under the ground: a loop at the nearest pool with oil left, never a dry one, a bloop every 5-20 s and a thicker gurgle every 30-70 s (the gurgle CC-BY, spookymodem - the first attribution a sound carries here). The interface clicks (Kenney, CC0): the menu's open and its two picks, the cards' buttons, the badge, the squads' calls, the trash can's presses. The loops are synthesized by `tools/soundgen` (stdlib Go to WAV, ffmpeg to OGG) so they repeat without a seam; everything short is a recording except the burst, made in code. View only, beside the lights it mirrors; `golib test` pinned nothing (it is golib's randomness), and a scripted shot checked it - in a shot the files are read and nothing is heard.
- 2026-09-22: the introduction tuned by its first play, on four counts. The **infrastructure** drop answers the colony's own work now - it comes in with the **first delivery** home (`State.Deliveries`, incremented where a robot's arms empty into the stores), not on a clock, so the first badge lands while the haul is still the whole game and no minute sits idle. The **guard post** answers the drawing of the scout's mark becoming **inevitable** - a rival drinking at the tanks (any `Enemy.Oil` over zero), or the mark once it lies on the ground (`len(Marks) > 0`) - not the party's driving in and not the scout's leaving: the badge lands while the thief is still siphoning, the post stands before the next visit, and its callout says so. The menu and the cards **offer only what a click would really do**: what the schematics, the ground, the fog or the stores refuse is gone from the rings and off the panels (`build pump`, `lay pipe`, `build trooper`), dimmed options were a spoiler the game no longer needs - a group with nothing to raise doesn't appear, and a ring that empties while open puts the menu away (`updateRadial`). And the deposits' organic bodies obey the eye: a cell a pool's or vein's ore doesn't cover builds like any other (`oreAt` on the generator's per-cell table, `buildableGround` shared by `canPlace` and the menu's cell), where the ore draws only the pump stands. Pinned by `tech_test.go` (the delivery and driving-party triggers, the hidden buttons), `radial_test.go` (the rings shrink to the raiseable) and `buildings_test.go` (`TestABuildingStandsWhereADepositsOreDoesntCover`); a shot checked the guard's callout.
- 2026-09-22: the buildings arrive as **remote schematics** (`sim_tech.go`, `tech.go`, `tech_test.go`), the introduction's first ladder, ridden as the difficulty saw. Six drops: infrastructure (silo, warehouse, charger) at half the way to the scout (`techInfraTicks`, derived), the guard post alone after the scout's visit (`Raids.Visits` 1), the frontier kit (protector, pump, and with them the pipes) at `techFrontierTicks` 5:30, the factory at `techIndustryTicks` 7:00, the war factory after the first raid (`Visits` 2), artillery when a base settles (`settled`). Arrival is derived from the state and never stored; only "opened" is (`State.Tech`, written by `AckTech`), so a save with the badge unclicked keeps it and an old save wakes with what it earned, opened, and owes no clicks. A drop waits as a badge over the monolith - breathing halos in the drop's ink around the group's mark or the blueprint's body - while the HUD says `schematics at the core`; the click opens a callout anchored to it and the next click puts it away. What hasn't arrived is nowhere: the radial leaves locked kinds and empty groups out (the menu doesn't open at all before the first drop), `build pump` and `lay pipe` stand dim (`LayPipe` asks for the frontier kit by name), and `MarkBuilding` refuses what hasn't arrived. Dev tools: `next schematics` (`DevNextTech`). Pinned by `tech_test.go` (the clock and the visit triggers, the refusal of `MarkBuilding` and `LayPipe`, the ack, the old save's migration, the round trip, the dimmed buttons) and `radial_test.go`'s filtered rings; shots checked the badge, the callout and the menu that opens with one group. The old ladder note in the introduction ("no clock, silo at 80% of the roof, frontier after the mark, guard when they camp") is retired to this entry: the clock of the first tooth is on purpose, and the roof-triggered storage stays a candidate for the measured loop.
- 2026-09-22: the build menu goes hierarchical and gets icons. It is two rings deep now: the build groups first (industry, military, logistics), then the group's own blueprints, so eight options become three and then two or three. A right click goes back a ring and closes the menu from the first. Every option wears an icon, and a blueprint's icon is the very body the region draws (`drawBuilding`), scaled down into the circle, so a change to a building's look is a change to its icon too; the groups, which stand for no one building, carry marks of their own in `glyphs.go`. Pinned by `radial_test.go` (every blueprint in exactly one group, the ring's geometry, the back-and-close of a right click); scripted shots opened the menu on a cell near the core, drilled into industry and checked the three marks and the two miniatures inside their rings.
- 2026-09-22: economy measurement and the first technology ladder. `economy_test.go` runs the real simulation headlessly for one hour over safe harvesting, worker growth and a protected oil outpost, on seeds 0, 1 and 2, and writes a minute-by-minute CSV only when `NIEBLA_ECONOMY_REPORT` names its path. The initial report found the safe loop's 60 L and 40 kg per minute, the oil roof at about minute 13 and the lilac roof at about minute 44; the outpost has a protector at minute 2, pump at 5, guard at 11 and laid pipe at 17. No dial changed: those observations now decide the next reports. The introduction has a proposed milestone ladder in the prototype scope, shaped as a difficulty saw: harvest, workers, storage, frontier, guard, squads, artillery. It also exposes the factory's current false promise: extra workers cannot raise extraction while a deposit takes one robot, so the worker-task and multi-worker design comes before a real unlock system.
- 2026-09-21: the squads' strip became the squads' **boxes**, one per squad at the top right: a little tank drawn in shapes (barrel, turret, hull, tracked wheels, `drawTankIcon`), the troopers the squad counts beside it and, below, the key that calls it; an empty squad's box wears a dim icon and its 0, the ordered squad's box is ringed in green, hovering lights it like a dev button, and a click on a box calls the squad as its key does (`updateSquadBoxes`, taken before the region hears of the click). Pinned by a box-geometry test (three boxes lie apart on the screen and pick their slot); shots checked the boxes at rest, the tank close up at scale 3, and the click on a box arming its squad.
- 2026-09-21: squads answer to numbers, and a counter-order stops being a hunt. The number keys 1-9 call a squad - 1 the first war factory raised, 2 the next, a factory still building its first troopers included, so a key never shifts (`squadSlots`) - arming the same ordering pointer the card's `give order` does; the same key again puts it away. A strip at the top right lists the squads by key and by what they are at, the one being ordered lit, and a click on a squad's pennant or ring picks it where it stands (`squadMarkAt`), so a rally that fell on a vein is moved without finding the factory. One click still gives the order: a rival vehicle to attack its party, the ground to guard it. Pinned by the slot and mark-picking tests; scripted shots from a state with two squads (`shotstate_test.go`, `NIEBLA_SHOT_STATE`) checked the armed strip and help, the rally moved by key and click, the key's coming back up and the pennant's arming.
- 2026-09-21: the settled enemy, artillery, and shots that fly (`sim_shots.go`, `shots.go`, `shots_test.go`), the third slice of enemies and battles. From the third visit on a party comes to stay: its crawler digs in as a base that grows a level every 2.5 min and from level 2 shells the colony's buildings, which can be hurt now, fall into a pile of half their cost and are mended by the robots. The colony's artillery, an eighth blueprint, shells what somebody of the colony sees, for lilac and oil a shell. Nothing hits at once any more: guard posts, troopers and the rivals' guns fire bullets that fly to their target, artillery lobs shells that burst where they were aimed, and all of it is state. On the screen the lasers are gone: streaks and arcs over their shadows, pools of light added over ground, buildings and mist as a shot passes, flashes at the guns, and bursts of sparks that cool from yellow to red, embers and smoke. Dev tools: `rivals: a base`, and `rivals: stop waiting` grows a base too. Pinned by `shots_test.go`; shots from a seeded duel checked two shells in the air with their pools and shadows, a burst on the colony's piece at three moments - flash and ring, sparks in the air, embers and smoke -, and the base with its gun.
- 2026-09-21: two robot traps found in a played save, and the tank's law changed. **The refill trap**: a robot burned a drop before asking whether its tank was full, so at the refill post it never was, and its first refill parked it there for good - every built robot of the save stood on the core or the charger with 120 L, which is why nobody went for the deposits out west (`SendRobot` kept picking a parked robot as the nearest free one). **The tank now burns only while the robot carries something** - walking out, loading, resting and refilling are free -, full is `tankFullSlack` short of the brim, and a dry post holds nobody with oil left in its tank, since the way to fill the posts is to go and fetch oil. **Stale posts**: a post with nothing left to give - dry under another's hands, or on ground an old save names and the generator has since moved - releases its robot on the next tick instead of leaving it idle and taken. Pinned by `TestARefueledRobotGoesBackToWork`; a copy of the save run forward showed the four parked robots leaving and a robot sent west working its pool. Dev tools: `rivals: stop waiting` under the rivals' button, which sends a camped party in at once, and `fast forward x8` under `new world`. The news plate moved down to clear it.
- 2026-09-21: squads land (`sim_squads.go`, `squads.go`, `squads_test.go`), the second slice of enemies and battles. The war factory, a seventh blueprint, builds troopers - combat robots that do no work - and the troopers it builds are its squad, up to six, ordered as one from its card: `give order`, then one click on a rival vehicle (attack its party, that vehicle first) or on the ground (guard that spot). A pennant marks a guarded spot and a ring the vehicle to shoot first. Troopers shoot out of their own tanks, the rivals shoot back at them and only at them, a fallen trooper leaves a wreck, a resting one mends under a bubble, and a squad with nobody left to attack goes back to its door. It is the first time the player gives a direct order. Pinned by `squads_test.go`; shots from seeded states checked five troopers closing on a camped crawler inside its clear pocket with the ring on it, the war factory's card with its two buttons, and an order scripted through the button: the `guard here` label by the pointer, then the pennant and the squad on its way.
- 2026-09-21: clear air shows as clear air (`mist.go`). The circles lost their blue tint and the fog its tiles: inside a repulsor's circle the ground shows as it is, a haze hangs over everything outside the circles, and past the line it thickens in 16 round layers to whole fog. The layers leave the circles out - the core's, the protectors', the rivals' repulsors - so a protector deep in the mist is a round window onto its ground, in a swell too, and a rival party is a hole in the fog, literally. The swell's crests stop at the circles; the ground is drawn a little farther out and under far protectors, since the mist no longer hides its stepped edge. Shots checked the region far out, two protectors in the mist under a held swell at two zooms, and the front at the closest stop; 1500 frames close up on the front ran in under 8 s.
- 2026-09-21: the rivals land, the first slice of enemies and battles (`sim_enemies.go`, `enemies.go`, `enemies_test.go`), designed the same day in [The rivals](#the-rivals): people in vehicles who come for the oil. A scout four minutes in siphons 25 L, sprays its mark on the ground and leaves, and the news tell the player what it did and what comes next; then raids - a crawler carrying the party's repulsor and its raiders - camp at the edge of the clear ground, get ready while the HUD counts it down, drive to the nearest tank with oil, fill up and go, each raid bigger and quicker. The fog digests a vehicle with no repulsor over it, so a dead crawler takes its party with it; wrecks drop a little loot and all they stole as a pile. The colony answers with a sixth blueprint, the guard post, which shoots for oil and shows its reach when picked. The state got its own PRNG (`State.roll`). Dev tools: `rivals: next visit`. Nobody damages the colony yet; squads and the settled enemy with artillery follow. Pinned by `enemies_test.go`; shots from seeded states checked the camp at the mist's edge with its countdown and news, a raid of five under a guard post's fire close up, and the scout leaving under its pocket with its mark on the core's pad.
- 2026-09-21: the prototype gets a finish line, [Prototype scope](#prototype-scope): enemies and battles, economy analysis and balance (with the difficulty saw as its reference), an introduction that unlocks the game little by little, humans as buildings and at least one more resource, and electricity with a wireless grid robots coordinate through. Headings only; nothing in the code changes.
- 2026-09-21: the ground is generated (`worldgen.go`, `ground.go`, `worldgen_test.go`), and the hand-made layout is gone. `State.Seed` names the region, `land` is the ground of the seed in hand (`useRegion`, which `Apply` calls), and a save from before wakes up on seed 0 with the deposits it doesn't know full (`State.enterRegion`). **Relief** by wave function collapse: terraces a level (4 m) high around great plains, heights on the cells' corners, `project` lifting whatever stands on the ground and `unproject` closing in on the ground under the pointer; buildings ask for a flat cell. **Cover**: zones of bare ground, lichen, grass and scrub, a grain per cell, slopes lit from the upper left, bushes in the scrub and rocks on the slopes from zoom 4, tufts and pebbles from zoom 8; the ground draws by blocks of 4, 2 or 1 cells by zoom, culled to the view, a few thousand triangles a frame. **Deposits** are fields of richness: painted cell by cell in layers of overlapping blots - stain, dark sheet, bright sheet; shelf and up to three crystals - so the middle is one body and the rim breaks into puddles and specks; a worked deposit wears from the rim in (`oreCut`), and they hold different amounts, the far ones more. The cards read `pool 8.3 ha`. Robots load by the deposit's heart (`postSpot`), the pump stands on it. Rocks and bushes are decoration and no longer hold a tile. **Dev tools**: `reset world` and `new world` (`DevResetWorld`). Shots checked stops 0 to 3 on seed 0 and a new world from the strip.
- 2026-09-21: five notes from play. **Pipes are laid by the section, by many hands**: a robot claims the nearest free section of the oldest unlaid pipe - the claim is state, `Robot.Pipe` and `Robot.Section`, which is how the others know -, stands by it for `pipeSectionWorkTicks` 120 (2 s, it was 1 s shared at the pipe's head, walking, which read as instant) and claims another; `Pipe.SectionLeft` holds the work by section, the laid sections draw among the ghost of the rest, and a half-laid pipe from an old save keeps its work. **Idle robots rest by the core**, in ranks of five before its broad face that close up when one leaves, with `N idle` written under them past ten or while far out; the 150 u ring is gone. **A load unloads at the core's side** when the core is its nearest store: it used to walk to its parking spot on the ring, up to 150 u past the core. **Stores wear a fill bar** on their body: silo, charger, warehouse, and two on the core. **The cell is the unit of picking**: the panel lists the picked cell alone, its outline replaces the tile's, and the header reads `cell c, r`; deposits and the core still show whole from any of their cells. Pinned by `TestRobotsLayAPipeASectionEachAndStandByIt`, `TestAHalfLaidPipeFromAnOldSaveKeepsItsWork`, `TestIdleRobotsRestInRanksByTheCore`, `TestALoadEndsItsWalkAtTheNearestStore` and the panel tests moved to cells; shots from seeded states checked fourteen robots spread along a pipe's sections, the ranks with `14 idle`, the bars, and a silo picked with a charger beside it.
- 2026-09-21: oil gets a place, and pipes become a network. The one oil stock is gone: oil sits in tanks (`sim_oil.go`: the core's, each silo's, each charger's 200 L), robots pour their load into the nearest tank with room and refill from the nearest charger or core with oil, payments come out of any tank, and a demolished tank drops its own oil. Pipes run from a pump or a tank into a tank, three to a building (six to the core), one between two ends, 4 L/s apiece, a source's pipes sharing what it gives; every end's card lists its pipes with a `remove` each and `lay pipe` while a port is free. Pipes stand on posts and cast a shadow. Fixed on the way: a pipe aimed at a silo beside the core ended at the core - a building now wins over the monolith, and the pointer's label names the tank before the click - and the radial's dimmed options, which asked `WithOpacity` for 80 instead of 0.3 and never dimmed. Lilac stays one stock. Pinned by `pipes_test.go` (`TestOilHasAPlaceAndPipesMoveItBetweenTanks`, `TestRobotsCarryOilToATankWithRoomAndRefillFromTheirPost`) and the old oil tests rewritten to the tank law; shots checked a pump-silo-silo-charger network flowing, its panel rows, the posts and shadow close up, and a pipe closed on a silo one tile from the core.
- 2026-09-21: closing a pipe gets easier, after the first play couldn't find how. A click on the pipe's last node opens a menu around it - `connect` (to the nearest oil store), `undo`, `cancel` -, the stores a pipe may end at wear a ring while the pointer is armed, and the click that ends a pipe is tested against the store's whole body on the screen, foot to top, where it used to ask for the foot alone - which on the core's 36 m monolith is not where anyone aims. A scripted shot opened the menu on the second bend and connected through it.
- 2026-09-21: pumps and pipes land (`sim_pipes.go`, `pipes.go`, `pipes_test.go`), and leave Later. The pump is a sixth building, the one that stands on oil, marked from its pool's card (`build pump`, `pumpCostLilac` 150 kg) and shown on every tile of the pool. `lay pipe` on its card arms the pointer: clicks bend the pipe, a click on a silo or the core ends it, a right click takes a bend back; the curve is a centripetal Catmull-Rom spline through the clicks, with a sag when there are none. It is paid by the section (5 kg per 25 m), laid by the robots from the pump out as the last part of their build line, and once laid it carries 2 L/s from the pool into the stores, with blobs of oil running down it, drawn over the fog where it crosses it. State gained `Pipes` and `BuildingPump`; actions gained `LayPipe` and `RemovePipe`, and `Demolish` takes a building's pipes with it, their cost in its pile. The two-pick, no-bends pipe of the old note lost to the player's own clicks. Pinned by `pipes_test.go`; shots from seeded states checked the panel's buttons, the pipe in hand with its price, the ghost of the part still to lay, the flow close up and a pipe wandering into the mist.
- 2026-09-21: the swell presses in little by little, and shows. The sim grew `Fog.Pressure`, which a swell pushes to 1 and the calm back to 0 over `fogSwellRampTicks` 300: the line, the pushed band and the extra burn follow it, where they used to jump with `SwellLeft`; the forecast stays exact, since a swell still starts at its cycle's end. `swell.go` draws it, by as much as it presses: crests of shade rolling in from far out and breaking on the line, and one-pixel static crawling over the mist, both a pure picture of the state's tick. The motes took their official name, **mites** (`mites.go`, every dial), and a note that people call them worse, which owes the game some people. Pinned by `TestASwellPressesInAndLetsGoLittleByLittle`; shots of a held swell checked the faint crests a third of the way in and the full ones.
- 2026-09-21: the mites' darkness goes radial and all the way to nothing: a square heart in a halo of discs, `miteRings` 6, the halo from `miteHaloOpacity` 0.2 easing to clear at the rim - the usual spark, in darkening. It used to stop at 0.1 on a square edge. A close-up shot checked the halo.
- 2026-09-21: the dev tools (`dev.go`, `dev_test.go`): Control and two clicks on the game's name open a strip with two buttons, one that holds a swell up until it is let go (`DevHoldSwell`, `Fog.Held`; it doesn't count among the fog's swells) and one that arms the pointer to place free built robots anywhere in the region (`DevSpawnRobot`). Both are actions, so the state still has one door. The inverse of `project` got a name, `unitsAtWorld`, which `cellAtWorld` now uses. Pinned by `dev_test.go`; a scripted shot opened the strip, held a swell - the line pressed in, the HUD said `swell` - and dropped two robots in the mist, where their mites gave them away.
- 2026-09-21: the fog's wear shows (`mites.go`, `mites_test.go`): mites of darkness orbit whatever stands in the mist, a few per cubic meter of its body - `mitesPerCubicUnit` 0.04, so a robot wears 7, a pile 47, a protector's site 61, never more than `mitesMaxPerHost` 400 - each a black heart with a falloff of 0.2 and 0.1 around it (`miteFalloff`), in black over normal blending, which takes light away. They follow their orbit `miteLagSeconds` 0.6 late, so a walker trails its swarm (`miteWideOrbit` 3 body radii), and what stands still for `miteGripSeconds` 2 gets it closed onto the body (`miteTightOrbit` 1); a host that leaves the state keeps its swarm for the `miteFadeSeconds` 0.5 of its fade. Far out they ring the icon, thinned, instead of the speck. View only: the play scene owns the field and the state never hears of it. Pinned by `mites_test.go`; shots from a seeded state checked a walker's trail, a builder's and a pile's tight swarm, the site's, and the far view. Oil pipes, the other candidate, went to Later as debt, with three complaints about micromanaging robots.
- 2026-09-21: the core stands up: the ring of light that read as a letter O is a monolith now, 16 by 4 m and 36 m tall (1 by 4 by 9), dark, with a lit top and a seam of light down its broad face. It draws with the buildings, back to front and never under their icon size, and the pad went down to the ground, so robots cross it instead of vanishing under it. `isoBox` grew a sibling for footprints that aren't square, `isoSlab`. The card says `height 36 m` and `monolith 16 by 4 m`. Shots checked it at stops 0, 3 and 5.
- 2026-09-21: demolition and loose items land (`sim_piles.go`, `piles_test.go`), as designed: the trash can on a building's or a site's card, pressed twice; `Demolish` and `CancelJob`; the pile on the cell with its card (`Loose items`), drawn as crates for the lilac and a drum for the oil, over the fog when no bubble covers it; robots fetching piles after build jobs and before their posts, never loading what the stores have no room for; every load, from a pile or a deposit, unloading at the nearest store of its kind. The site got a card of its own (`Building site`: what rises, the work left, what was paid). The robot's day became a list of tasks (`robotDay`) that both the rules and the captions read. Pinned by `piles_test.go`; a scripted shot from a seeded state armed a silo's can (red, `demolish?`), pressed it again and saw the silo gone, its 100 kg on its cell and the oil roof back to 1.0 kL.
- 2026-09-21: decided, robots are generalists; specialization comes as a per-robot task list the player edits, noted under Later as debt for the UI pass. Nothing in the code changes yet.
- 2026-09-21: demolition and loose items designed on paper (yet to land): a trash can button on a building's or a site's card, pressed twice, takes it down at once - its tasks cancelled, its whole cost plus the stock its roof was holding left on its cell as one pile, a massless container that stands for all of it and vanishes with its last item. The pile won over scattering the items across the footprint: one entity, one card, readable at every zoom. Robots clear piles after build jobs and before their own posts, and every load now ends its walk at the nearest store of its kind - warehouse or core for solids, silo or core for liquids. A protector can't be demolished while it alone shelters another building.
- 2026-09-20: the lifecycle, and the player's name on the door: a menu scene (`menu.go`) - the game's name, the player's number (`#30E99076` style), Play and Quit - and Play carries the player to their base as they left it. Identity (`identity.go`): the machine's system ID (MachineGuid / IOPlatformUUID / `/etc/machine-id`), salted and hashed, is the player - shown as its first eight, never raw; a machine with no ID gets a random one kept in the database. The local database lands (`store.go`, SQLite through `modernc.org/sqlite`, pure Go): `players` (identity, source, created_at, a `token` column waiting for the server), `saves` (the whole State as JSON per player and slot) and `machine`; the file lives at the settings folder in `GoLib games/niebla/`, so debug and dist builds share the base, and under `golib shot` it is `:memory:` so shots never touch it (and `--save` still works, taking priority). Esc in the region now saves and returns to the menu instead of quitting; autosave every 15 s covers a window closed without ceremony. The monitor filters moved from the play scene to `main`, made once, so the menu runs under them too. Pinned by `identity_test.go` and `store_test.go`; shots checked the menu (scaled) and the region reached with `Enter@1`; a real run against a redirected settings folder registered the player row, and its ID matched the menu's number.
- 2026-09-20: created with `golib new` (skeleton square). Design brief written from the design talks: gray fog survival, hybrid nomad/sedentary, oil + lilac, robots, indestructible core, mark-to-build, serializable state with action reducers, 2K isometric.
- 2026-09-20: slice 1, the painted region: 25x25 hand-made layout in code (`region.go`), isometric projection, oil pools, lilac veins, the core with its bubble, a static fog line (`draw.go`). Skeleton square removed; nothing moves yet. All text English; JSON string files wait for the first text-heavy screen.
- 2026-09-20: slice 1 polished after review: screen at 2K/2 (1280x720, `PixelArt`); the white slivers at the fog border gone (the fog fades in per tile now, no band); two generous seams of oil and lilac cross under the core, inside the bubble; robot and building-cost policy written down for the next slice.
- 2026-09-20: a subtle monitor filter, `shaders/soft.fs`, rounds the pixels' corners (F2 turns it off); flat areas and the ground's checker stay untouched, and no grain is added.
- 2026-09-20: the filter becomes the asteroids pair, turned down: `glow.fs` with its threshold at 0.8 (the pale fog outshines the oil, so only the core, the bubble's edge and the text glow) at strength 0.9, and `crt.fs` without flicker, scanlines at 0.96, a lighter vignette and curvature 0.08. The soft rounding runs last.
- 2026-09-20: slice 2, the camera: WASD, arrows or left stick pan, and the wheel zooms in whole steps from 1 to 4, anchored on the cursor, the view bounded to the region. The world took its unit (`unitsPerTile` 5, with the proportions it pins: robot 1 u, pole 5 u, building 10 u, vein 30 u), `project` speaks units, and the core's pole now fills its whole tile. Rocks and bushes dot the ground from zoom 2 on, off their tile's middle by a stable jitter. The cell-and-offset model is decided and written down, waiting for the robots.
- 2026-09-20: first feel pass: the zoom glides between its whole steps instead of jumping (`zoomGlide`), still anchored on the cursor, and holding the right button drags the view like a grabbed map, both working together.
- 2026-09-20: slice 3, tile inspection: a click picks a tile (right-click cancels, a drag never does) and a panel lists what stands there, one card per thing, headlined by its amount, expandable on click to its details. The entity database lands in `catalog.go` (name, color, unit, card lines per type; a stable hashed color for types it has no entry for), the `[name]...[/]` color markup in `markup.go`, the panel in `inspect.go`. The world took its SI units (1 u = 1 m; oil in L, lilac in kg) in `things.go`; the two seams and the core have fichas. Technical notes moved to [README.md](./README.md).
- 2026-09-20: slice 4, the robots - the simulation lands (`state.go`, `actions.go`, `sim_robots.go`, `world_test.go`): the core starts with two idle robots; an expanded oil or lilac card has a send robot button (recall robot once the tile has its robot), and the nearest free robot takes the post, hauls loads home and is released when the deposit runs dry, leaving a scar. Build jobs sit in the state and already pull a robot home before its own post, though nothing marks them yet. Stores show in the HUD and the core's card; robots are drawn (body, lit top, glowing eye, cargo pack) and carded with what they are doing. Verified by tests and shots: determinism, JSON round trip, haul conservation, job priority, dry release.
- 2026-09-20: the R.U.S.E. rescale: the world grew from 125 m to 5 km across (`unitsPerTile` 5 to 200), the wheel now jumps between stops that double the zoom (1 to 32, gliding, anchored on the cursor), deposits became 4-tile patches - one of each kind inside the bubble, two of each kind 1.5 to 2 km out - and the in-between ground stays empty but for world-sized rocks and bushes from zoom 4. Robots (6 u rovers at 12 u/s, parked on a 150 u ring) and the core's glow draw as points while far out; the bubble, fog line and tile highlights keep their thickness on the screen, not in the world. The per-tile amounts stay gameplay dials (900 L, 300 kg) so the hauling rhythm survives. Shots at stops 0, 3 and 5 checked the icons, the middle ground and the empty close-up.
- 2026-09-20: the ant rhythm: robots run at 30 u/s but carry 30 L / 20 kg, so a worked deposit shows a constant coming and going of small loads. Veins and pools became single things: each patch of four tiles is one unit - one card on any of its tiles (the safe vein reads 12.0 t: `lilacPerVeinTile` went 300 to 3000 kg), one robot per patch (a send on another of its tiles changes nothing), what remains in `State.Drain` under one key per patch (`Deposit` in `region.go`, flooded once out of the layout). Graphically each patch draws as one continuous body - a pool as a single sheet, a vein as a shelf with crystals over its tiles - that shrinks as it drains and leaves one big scar when dry. Patch law pinned by `TestDepositPatchIsOneUnit`; shots checked the continuous bodies and the patch card.
- 2026-09-20: cards open by themselves: a tile's primary thing (deposits, buildings, the core — `Primary` in the catalog) and any thing alone on its tile start expanded, so one click on a vein shows its amount and its send robot button. A click still folds or reopens, remembered per card.
- 2026-09-20: the buildings, and with them the colony's second robot: five blueprints marked with keys 1-5 (factory, charger, silo, warehouse, shadow protector), paid up front and raised by the robots' job queue; nothing may stand outside a bubble but a protector, whose small bubble shelters the rest. The factory builds robots from lilac and oil (the way collection scales: more robots, more parallel hauling); a built robot burns oil from its tank, walks itself to the nearest charger or the core when it runs low, and is digested at zero outside a bubble. The fog now drags at every walker's pace and the bubbles cancel it. Silos and warehouses give the stores a roof the robots wait under when it fills. State gained `Buildings` and `Job.Kind`; actions gained `MarkBuilding` and `QueueRobot`. Pinned by `buildings_test.go`; shots checked the marking ghost, the five bodies at rest zoom and close up, the factory card's build robot button and the new robot's rollout.
- 2026-09-20: the grid subdivides, and building goes radial. The tile's last subdivision is the cell, 40 u on a side (5 by 5 to a tile): buildings sit on cells, several to a tile, and the marking ghost is a cell-sized diamond. A click on empty ground opens a radial build menu around the tile (radial.go) - the keys 1-5 stay as shortcuts - and a site under construction wears a wireframe of the body to come with its progress bar, drawn over the fog. canPlace now refuses a cell with a job pending, pinned by TestCellsFitSeveralBuildingsToATile; shots checked the radial, the ghost, the wireframe and a factory and charger sharing one tile. Debt noted: deposits and the drawn ground stay tile-shaped; making veins read as irregular continuous bodies over the cell grid comes later.
- 2026-09-20: the cell tightens to 25 u (8 by 8 to a tile, four robots across - the old 40 u read more house than the colony builds), the bodies rescale to their footprint, and the buildings become buildable from minute one: the core starts with a gift of 300 L and 600 kg, the ghost turns red when the stores can't pay, and the radial menu dims the blueprints they can't. The scaffold now tells the story: the built part rises from the ground in solid colors inside the wireframe of the whole body, and the builders stand on their cell's edge, spread by ID, where the rising body can't swallow them. What the first playthrough missed - markings silently refused with empty stores - is what the gift and the honest ghost answer; shots checked the rise from wireframe to factory-plus-charger on one tile.
- 2026-09-20: the cursor stops masquerading as a tile: it is the cell, always - a small diamond under the pointer, lifted to a readable size when the view is far out. The old full-tile hover highlight is gone; the picked tile keeps its outline. Shots checked the cursor at rest zoom, the green fill while marking, and the builders ringing their site.
- 2026-09-20: building loses a step, and the panel wins its ground. The build menu now opens on the clicked cell itself (it used to anchor on the tile's center), its options are ready or dimmed per what that cell and the stores allow, and picking one raises the building right there - no ghost, no second click, no marking mode; the 1-5 keys and the palette line are gone with it. The inspection panel also takes precedence over what sits under it, so a button keeps working when the panel covers buildable ground - it used to open the build menu under the pointer. Shots checked the menu anchored off a tile's center and a send robot pressed through a panel over ground.
- 2026-09-20: the fog designed on paper (slice 5, yet to land): it breathes in swells instead of creeping on the core - the core's bubble is never touched, conflict is timing rather than turrets, and outposts become storm shelters. Going out into the mist is sketched as the robot's adventure: expeditions, for later.
- 2026-09-20: slice 5, the fog breathes (`sim_fog.go`, `fog_test.go`): whole cycles of calm (30 s each), then a swell rises at a cycle's end - the line presses in, the pushed band drags at a quarter of the step and built robots outside a bubble burn 1.5x - telegraphed a cycle ahead on the HUD and as a ghost line on the ground, in a stronger hand while it lasts, and each swell sooner, longer and deeper than the last, never past the bubble's margin. The weather is state (`State.Fog`), so saves and replays carry it; the replay test crosses the first swell. Pinned by `fog_test.go`; shots checked the calm line, the forecast's ghost, the pressed-in swell over the far deposits and the HUD's words.
- 2026-09-20: fix: a protector's bubble drew at its cell's index read as tiles - eight times too far, off the region - so a raised post showed no protected radius around it. The shelter itself always worked (`inSafeZone` reads cells); the drawing now centers the pocket on the building's ground cell (`protectorBubbleCenter`), pinned by `TestAProtectorsBubbleStandsOnItsCell`, and a scripted shot raised a protector outside the bubble and saw its pocket around it.
