# Help for ~macros~

~macros~ are special aliases that can issue one or more commands.

## Usage:

  ~set =# [command]~ - e.g. ~set =1 "say hi everyone"~  
  Sets a macro # (0-10) to a command of your choice. You can then use this macro  
  as a shortcut to quickly issue to a command as follows: ~=1~  
  If your terminal program supports it, prettying a corresponding F-Key
  
  ~set =#~ - e.g. ~set =1~  
  Clears a macro # (0-10)

  ~=#~ - e.g. ~set =1~  
  Executes macro # (0-10)

  ~=?~  
  Lists all currently set macros.

## Special Usage:

  You can split a macro into more than one command by separating extra commands  
  with a semicolon - **;**  
  e.g. ~set =1 wave;say hi;emote sits down~

  **See also:** ~help set~