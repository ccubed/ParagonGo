
local nouns = {"quest", "hunger", "hungry", "belly", "food"}

--- Called when a user issues a command to or near the mob.
-- @param cmd string - The command issued.
-- @param rest string - The arguments following the command.
-- @param mob ActorObject - The mob.
-- @param room RoomObject - The room the mob is in.
-- @param eventDetails CommandEventDetails - Additional event context.
-- @return boolean Return true if the event was handled.
function onCommand(cmd, rest, mob, room, eventDetails)
    if cmd == "wave" then
        mob:Command("wave")
    end
    return false
end

--- Called when a user asks the mob a question.
-- @param mob ActorObject - The mob.
-- @param room RoomObject - The room the mob is in.
-- @param eventDetails AskEventDetails - Details about the ask event.
-- @return boolean Return true if the event was handled.
function onAsk(mob, room, eventDetails)

    local user = GetUser(eventDetails.sourceId)
    if user == nil then
        return false
    end

    local match = UtilFindMatchIn(eventDetails.askText, nouns)
    if match.found then

        if not user:HasQuest("4-start") then

            mob:Command("emote rubs his belly.")
            mob:Command("say I forgot my lunch today, and I'm so hungry.")
            mob:Command("say Do you think you could find a cheese sandwich for me?")

            user:GetParty():GiveQuest("4-start")

        elseif user:HasQuest("4-end") then
            mob:Command("sayto @" .. tostring(user:UserId()) .. " Thanks, but you've done enough. Too much, really.")
        else
            mob:Command("emote rubs his belly.")
        end

        return true
    end

    return false
end

--- Called when a user gives the mob an item or gold.
-- @param mob ActorObject - The mob.
-- @param room RoomObject - The room the mob is in.
-- @param eventDetails GiveEventDetails - Details about the give event.
-- @return boolean Return true if the event was handled.
function onGive(mob, room, eventDetails)

    if eventDetails.sourceType == "mob" then
        return false
    end

    if eventDetails.gold > 0 then
        mob:Command("say I don't need your money... but I'll take it!")

        -- Check a random number
        if RandInt(1, 100) > 50 then
            mob:Command("emote flips a coin into the air and catches it!")
        else
            mob:Command("emote flips a coin into the air and misses the catch!")
            mob:Command("drop 1 gold")
        end
        return true
    end

    if eventDetails.item.ItemId ~= 0 then
        if eventDetails.item.ItemId ~= 30004 then
            mob:Command("look !" .. tostring(eventDetails.item.ItemId))
            mob:Command("drop !" .. tostring(eventDetails.item.ItemId), UtilGetSecondsToTurns(5))
            return true
        end
    end

    local user = GetUser(eventDetails.sourceId)
    if user == nil then
        return false
    end

    if user:HasQuest("4-start") then

        user:GetParty():GiveQuest("4-end")
        mob:Command("say Thanks! I can get on with my day now.")
        mob:Command("eat !" .. tostring(eventDetails.item.ItemId))

        return true
    end

    return false
end

--- Called each round when the mob is idle.
-- @param mob ActorObject - The mob.
-- @param room RoomObject - The room the mob is in.
-- @return boolean Return true if the event was handled.
function onIdle(mob, room)

    local round = UtilGetRoundNumber()

    local grumbled = false
    local allPlayers = room:GetPlayers()

    -- playersTold maps stringified userId -> round number. String keys are used
    -- so the table survives the round-trip through GetTempData/SetTempData.
    -- GetTempData returns the stored value, so copy it into a native Lua table
    -- before indexing, iterating, or mutating it.
    local playersTold = {}
    local stored = mob:GetTempData("playersTold")
    if stored ~= nil then
        for k, v in pairs(stored) do
            playersTold[k] = v
        end
    end

    if #allPlayers > 0 then

        for i = 1, #allPlayers do

            local uid = tostring(allPlayers[i]:UserId())

            if playersTold[uid] ~= nil then
                if round < playersTold[uid] then
                    goto continue
                end
            end

            if not allPlayers[i]:HasQuest("4-start") then
                if not grumbled then
                    mob:Command("emote pats his belly as it grumbles.")
                    grumbled = true
                end
                mob:Command("sayto @" .. uid .. " I'm so hungry.")
            else
                playersTold[uid] = round + 500
            end

            playersTold[uid] = round + 5
            -- Don't need to repeat to every player.
            do break end

            ::continue::
        end

        if next(playersTold) ~= nil then
            mob:SetTempData("playersTold", playersTold)
        else
            mob:SetTempData("playersTold", nil)
        end

        return true
    end

    for key, value in pairs(playersTold) do
        if value < round - 100 then
            playersTold[key] = nil
        end
    end

    if next(playersTold) == nil then
        mob:SetTempData("playersTold", nil)
    else
        mob:SetTempData("playersTold", playersTold)
    end

    local action = round % 3

    if action == 0 then
        mob:Command("wander")
        return true
    end

    return false
end
