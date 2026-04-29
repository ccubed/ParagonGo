# Help for ~picklock~

The ~picklock~ command attempts to pick the lock of any exit or container.  
You'll need a set of *lockpicks*.

## Usage:

  ~picklock [exit name/container name]~ - This picks the lock. You can also use it to inspect the lock and select `quit` without attempting to pick it.

## Examples:

    **[HP:66/66 MP:35/35]:** ~north~  
    There's a lock preventing you from going that way. You'll need a *Key* or to ~pick~ the lock with *lockpicks*.


    **[HP:66/66 MP:35/35]:** ~pick north~

    <ansi fg="black-bold">.:</ansi> <ansi fg="table-title">The Lock Sequence Looks like:</ansi>

    <ansi fg="yellow-bold">╔═══════╦═══════╦═══════╦═══════╦═══════╗  
    ║       ║       ║       ║       ║       ║  
    <ansi fg="yellow-bold">║</ansi>   <ansi fg="red-bold">?</ansi>   <ansi fg="yellow-bold">║</ansi>   <ansi fg="red-bold">?</ansi>   <ansi fg="yellow-bold">║</ansi>   <ansi fg="red-bold">?</ansi>   <ansi fg="yellow-bold">║</ansi>   <ansi fg="red-bold">?</ansi>   <ansi fg="yellow-bold">║</ansi>   <ansi fg="red-bold">?</ansi>   <ansi fg="yellow-bold">║</ansi>  
    <ansi fg="yellow-bold">║       ║       ║       ║       ║       ║</ansi>  
    ╚═══════╩═══════╩═══════╩═══════╩═══════╝</ansi>
    
    <ansi fg="black-bold">.:</ansi> <ansi fg="yellow-bold">Move your lockpick?</ansi> <ansi fg="black-bold">[</ansi>UP<ansi fg="black-bold">/</ansi>DOWN<ansi fg="black-bold">/</ansi>quit<ansi fg="black-bold">]</ansi>

**Note:** Each **?** symbol represents an UP or DOWN you must guess in the sequence.

**See also:** ~help picklock-example~, ~help keyring~