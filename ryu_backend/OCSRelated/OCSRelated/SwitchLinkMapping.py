fatTree = [(31,77), (32,78), (73,79), (74,80), #pod1 edge links
            (39,85), (40,86), (81,87), (82,88), #pod2 edge links
            (15,25), (26,65), (16,27), (28,66), #pod1 agg links
            (23,33), (24,35), (34,69), (36,70), #pod2 agg links
            (1,9), (3,10), (7,11), (12,61),     #pod1 core links
            (2,17), (4,18), (8,19), (20,62),   #pod2 core links
           (5,5), (6,6), (63,63), (64,64),
           (13,13), (14,14), (21,21), (22,22),
           (67,67), (68,68), (71,71), (72,72),
           (29,29), (30,30), (37,37), (38,38),
           (75,75), (76,76), (83,83), (84,84)]

#(physicalID, physicalPort) -> (logicalSwitchId, logicalGroupId)
def SwitchIDMapping(phyID, phyPort):
    switchOffsetList = {4:0, 5:3, 2:6, 3:9, 1:12}
    switchOffset = switchOffsetList[phyID]
    if(phyID == 1):
        index = int((phyPort - 17)/2)
        logical_sid =  index + 12
    else:
        if(phyPort in [17, 18, 23, 24]):
            internalOffset = 0
        elif(phyPort in [19, 20, 25, 26]):
            internalOffset = 1
        elif(phyPort in [21, 22, 27, 28]):
            internalOffset = 2
        else:
            raise ValueError()
        logical_sid = switchOffset + internalOffset
    logical_groupId = int(logical_sid/3)
    return (logical_sid,  logical_groupId)

def CircuitReconfig(sid1, sid2):
    ret = []
    if((sid1,sid2) == (0, 2)):
        ret += [(75, 77), (76, 78), (29, 15), (30, 65), (31,31), (32,32), (25,25), (26,26)]

    elif((sid1,sid2) == (1, 2)):
        ret += [(75, 79), (76, 80), (29, 16), (30, 66), (73, 73), (74, 74), (27,27), (28, 28)]

    elif((sid1, sid2) == (6, 8)):
        ret += [(25, 67), (27, 68), (13, 1), (14, 3), (15, 15), (16, 16), (9, 9), (10,10)]

    elif((sid1, sid2) == (7, 8)):
        ret += [(26, 67), (28, 68), (7, 13), (14, 61), (65,65), (66, 66), (11, 11), (12, 12)]

    elif((sid1, sid2) == (12, 14)):
        ret += [(9, 5), (17, 6), (1, 1), (2, 2)]

    elif((sid1,sid2) == (14, 12)):
        ret +=[(9,1), (17,2), (5,5),  (6,6)]

    elif((sid1, sid2) == (13, 14)):
        ret += [(10, 5), (18, 6), (3, 3), (4, 4)]

    elif((sid1, sid2) == (3,5)):
        ret += [(85, 83), (86, 84), (23, 37), (69, 38), (39, 39), (40, 40), (33, 33), (34, 34)]

    elif((sid1, sid2) == (5,3)):
        ret += [(39, 85), (40, 86), (23, 33), (34, 69), (83, 83), (84, 84), (37, 37), (38, 38)]

    elif((sid1, sid2) == (4,5)):
        ret += [(83, 87), (84, 88), (24, 37), (38, 70), (81, 81), (82, 82), (35, 35), (36, 36)]

    elif((sid1, sid2) == (5,4)):
        ret += [(81, 87), (82, 88), (24, 35), (36, 70), (83, 83), (84, 84), (37, 37), (38, 38)]

    return ret

if __name__ == "__main__":
    for x in range(1,6):
        for y in range(17, 29):
            print(x, y, SwitchIDMapping(x,y))



