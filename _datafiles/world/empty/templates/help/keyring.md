# Help for ~keyring~

The ~keyring~ command tracks your current keys and picked locks.

Whenever you use a key, it is taken from your items and added to your keyring forever. You cannot lose a key once this happens.

Similarly, if you pick a lock, you will always remember the sequence to pick the lock. The lock will be instantaneously picked when you attempt to cross through an exit or open a container.

## Usage:

  ~keyring~ - This will list all keys and lockpick sequences you have.  
            It will tell you the Type, Location, exit or container name, and  
            as the lockpick sequence if applicable.

## Example:

    **[HP:66/66 MP:35/35]:** ~keyring~  
    **Your Keyring:**  
    **╒══════════╕══════════════════════════════╕═══════╕══════════╕**  
    **│ Type     │ Location                     │ Where │ Sequence │**  
    **└──────────┘──────────────────────────────┘───────┘──────────┘**  
    **│ Key      │ #110 Inside of the Catacombs │ west  │ -        │**  
    **│ Lockpick │ #784 Inside a Residence      │ chest │ D U U    │**  
    **└──────────┘──────────────────────────────┘───────┘──────────┘**

**See also:** ~help picklock~