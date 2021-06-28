tilespec_txt = """
# 01:w    floor   CC1
# 02    wall    CC1
# 03    ice     CC1
# 04    ice wall ne     CC1
# 05    ice wall se     CC1
# 06    ice wall nw     CC1
# 07    ice wall sw     CC1
# 08    water   CC1
# 09    fire    CC1
# 0A    force floor n   CC1
# 0B    force floor e   CC1
# 0C    force floor s   CC1
# 0D    force floor w   CC1
# 0E    green toggle wall       CC1
# 0F    green toggle floor      CC1
# 10:w    red teleport
# 11:w    blue teleport   CC1
# 12    yellow teleport
# 13    green teleport
# 14    exit    CC1
# 15    toxic floor
# 16,D,+        chip    CC1
# 17,D,+        dirt block      CC1
# 18,D,+        walker  CC1
# 19,D,+        glider  CC1
# 1A,D,+        ice block       CC1
# 1B,+          thin wall s     CC1
# 1C,+          thin wall e     CC1
# 1D,+          thin wall se    CC1
# 1E    gravel  CC1
# 1F    green button    CC1
# 20    blue button     CC1
# 21,D,+        tank    CC1
# 22    red door        CC1
# 23    blue door       CC1
# 24    yellow door     CC1
# 25    green door      CC1
# 26,+  red key CC1
# 27,+  blue key        CC1
# 28,+  yellow key      CC1
# 29,+  green key       CC1
# 2A,+  ic chip CC1
# 2B,+  extra chip      CC1
# 2C    chip socket     CC1
# 2D    popup wall      CC1
# 2E    invisible wall  CC1
# 2F    invisible wall (temp)   CC1
# 30    blue wall       CC1
# 31    blue floor      CC1
# 32    dirt    CC1
# 33,D,+        bug     CC1
# 34,D,+        centipede       CC1
# 35,D,+        ball    CC1
# 36,D,+        blob    CC1
# 37,D,+        red teeth       CC1
# 38,D,+        fireball        CC1
# 39    red button      CC1
# 3A    brown button    CC1
# 3B,+    ice boots     CC1
# 3C,+  magnet boots    CC1
# 3D,+  fire boots      CC1
# 3E,+  flippers        CC1
# 3F    boot thief      CC1
# 40,+  red bomb        CC1
# 41    open trap
# 42    trap    CC1
# 43:d    clone machine   CC1
# 44:d    clone machine
# 45    hint    CC1
# 46    force floor random      CC1
# 47    gray button
# 48    revolving door sw
# 49    revolving door nw
# 4A    revolving door ne
# 4B    revolving door se
# 4C,+  time bonus
# 4D,+  time toggle
# 4E:w    transmogrifier
# 4F:t    railroad
# 50:w    steel wall
# 51,+  time bomb
# 52,+  helmet
# 56,D,+    melinda
# 57,D,+    blue teeth
# 59,+  hiking boots
# 5A    male-only
# 5B    female-only
# 5C:g    logic gate
# 5E:w    pink button
# 5F    flame jet
# 60    flame jet
# 61    orange button
# 62,+  lightning
# 63,D,+    yellow tank
# 64    yellow tank button
# 65,D,+    chip mimic
# 66,D,+    melinda mimic
# 68,+  bowling ball
# 69,D,+ rover
# 6A,+  time down
# 6B:s    custom floor
# 6D,P,+    thin wall
# 6F,+  rr sign
# 70:s    custom wall
# 71:G    symbol
# 72    purple toggle floor
# 73    purple toggle wall
# 76,m,+  modifier
# 77,mm,+  modifier
# 78,mmmm,+  modifier
# 7A,+  10 point flag
# 7B,+  100 point flag
# 7C,+  1000 point flag
# 7D    green wall
# 7E    green floor
# 7F,+  no sign
# 80,+  double points flag
# 81,D,A,+    direction block
# 82,D,+  floor monster
# 83,+  green bomb
# 84,+  green chip
# 87:w    black button
# 88:w    off switch
# 89:w    on switch
# 8A    key thief
# 8B,D,+    ghost
# 8C,+  foil
# 8D    turtle
# 8E,+  secret eye
# 8F,+  treasure
# 90,+  speed boots
# 92,+  hook
# F1,D,+:c sokoban block
# F2:c  sokoban button
# F3:c  sokoban wall
"""

def parse_tilespecs(txt):
    tilespecs = {}
    for line in txt.strip().splitlines():
        cc1 = 'CC1' in line
        line = line.partition("CC1")[0].strip()
        _, spec, name = line.split(None, 2)
        spec, _, modtype = spec.partition(":")
        parts = spec.split(',')
        byte = int(parts[0], 16)
        tilespecs[byte] = byte, parts[1:], modtype, name
        if cc1:
            #CC1_TILES.add(name)
            pass
    return tilespecs

for id, extra, modtype, name in parse_tilespecs(tilespec_txt).values():
    assert 'D' not in extra or extra[0] == 'D', 'D not first in {}'.format((id, extra))
    if extra and extra[0] in ('m', 'mm', 'mmmm'):
        continue
    flags = []
    extra = list(extra)
    if 'D' in extra:
        flags.append('hasDir')
    if [x for x in extra if x not in 'D+']:
        flags.append('hasExtra')
    if '+' in extra:
        flags.append('hasLower')
    assert len(extra) == len(flags), "excess extra {}".format(extra)
    if modtype:
        modtype = "'"+modtype+"'"
    else:
        modtype = '0'
    print("\t%d: {ID: %#x, Mod: %s, Flags: %s, Name: \"%s\" }," % (id, id, modtype, '|'.join(flags) or 0, name))
