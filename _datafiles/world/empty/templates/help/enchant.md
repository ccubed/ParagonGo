# Help for ~enchant~ (skill)

The ~enchant~ skill embues objects with enhanced properties.

As you level up the ~enchant~ skill, it can apply a great enchantment.  
Your odds of success (and degree of succes) are influence by the Mysticism stat.

**Beware**, no item can be enchanted more than once!

## Usage:

(Lvl 1) ~enchant [item]~ Enchant a weapon with a damage bonus.  
(Lvl 2) ~enchant [item]~ Enchant equipment with a defensive bonus.  
(Lvl 3) ~enchant [item]~ Add a stat bonus to a weapon or equipment in addition to the above.  
(Lvl 4) ~unenchant/uncurse [item]~ Remove the enchantment or curse from any object.

Check the odds of it exploding before you enchant it with: ~enchant chance [item]~

Enchant bonuses are calculated as follows:
    - Damage Bonus - **SquareRoot( Mysticism )**
    - Defense Bonus - **SqareRoot( Mysticism )**
    - Random Status Bonus - **SqareRoot( Mysticism )**
    - You get 2 random stat bonus at SkillLevel 3, and another at SkillLevel 4
    
    Enchanted items have a **25%** chance of becoming cursed.
    Items that are enchanted have a **50 - (SkillLevel*10) + (NumberOfEnchantments*20) - (Mysticism/4)%** Chance to be destroyed