
/**
 * Called when a player fully dies, before any death penalties are applied. Use this to implement custom death behavior such as a revival.
 * @param {ActorObject} user The dying player.
 * @param {RoomObject} room The room the player is in.
 * @returns {boolean | void} Return true to abort the default death (no penalties, no corpse). The script is responsible for whatever happens instead, and must not synchronously trigger another death.
 */
function onDie(user, room) {

    if ( user.HasBuffFlag("revive-on-death") ) {
        user.SetHealth(user.GetHealthMax());
        user.SendText(`<ansi fg="mute-lblue">You are revived in a shower of magical sparks!</ansi>`);
        room.SendText(user.GetCharacterName(true) + `<ansi fg="mute-lblue"> is suddenly revived in a shower of sparks!</ansi>`, user.UserId());
        user.CancelBuffWithFlag("revive-on-death");
        return true;
    }

    return false;
}
