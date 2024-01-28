# Twitch2DiscordBridge-Go
[Twitch2DiscordBridge](https://github.com/kabessao/twitch2discordBridge) re-written in Golang

This is the new version of the bot, but now written in Golang.

"But why golang" you may ask. Well, one issue I was having with python was concurrency, and this new version uses a lot of it. Some you will notice and some you will not.


Bellow is a list of new features: 

## Multpile instances in one binary 

One of the problems I was having with the first version was running multiple instances of the bot, which I acchieved by using docker containers, or you could run the same python
code in a different folder. Well not anymore.

With this new version you can run multiple instances of the bot by creating different configuration files. All you need to do is end the file with `config.yaml`, and you're done.

Plus everything is hot reloadable. Meaning you don't need to restart the application every time you have to change something. The application detects creation/deletion/change in any configuration file, and acts accordingly.

Everything you need to know is well documented in the `config_example.yaml` file in this repository.

## Emotes Lookup Table

In the previous version the emotes lookup table was done in the configuration file, which was clunky to deal with. So this time the lookup table is in a separate csv file `emotes.csv`, which is optional to have, but if you create the file it'll be used to display emotes. This can be useful if you already have the same twitch emotes on discord.

The csv file needs to be comma separated, and each row should have the first column as the emote on twitch, and the second column the emote on discord.

Ex.: 

henyaHenyaheart, <:HenyaHeart:1107237396036190270>

This file is also Hot reloadable, which means again that you don't need to restart the bot to add/remove emotes to it.

## Dynamic Emote Grabber

I have no creativity for names, and that's the one I choose for this functionality. 

By using a Discord bot in a separate discord server, I can get the emotes from a twitch message, upload those emotes to that Discord server, send the messages with those emotes, and delete them afterwards. With this any emote that is not present in the look up file will still be displayed.

You can enable this by creating a configuration file called `discordBot.yaml`, listing the bot token and the id of the server. Then you have to enable it on each bot configuration (details of it is listed on the example file).

Why do I delete the emotes afterwards ? Well, because you can have only a limited amount of emotes in a single Discord server.

Do note though that Discord rate limits endpoints that gets used too much in a short time, and the emotes related api is one of the most affected. Because of that the bot will send the message regardless of having all of the emotes or not, and if it's succesful at getting the extra emotes it will edit the message with the new emotes. You will se the "(edited)" mark whenever that happens.


## Overheating 

Kinda weird to list this as a feature, but it is, and I'll explain why.

Discord likes to rate limit end points that are being used too much in a short amount of time, including webhook endpoints. When that happens the bot will wait the timeout informed by discord to try to send the message again, and that can snowball into hundreds of thousands of messages waiting to be sent. Doing one stress test it took 6 hours for all of the waiting messages to be processed.

Because of that I've implemented something akin to miniguns in games, where when you shoot it they get hotter, when you stop it cools off, and when you reach the limit heat allowed it stops working until it's decently cool. And that's exactly what the application does.

The application keeps track of how many messages haven't been processed yet, and when that reaches 50 it will refuse to try sending any more messages. After that number reaches 5 it resumes opperations.

(this should never happen if you aren't using `send_all_messages` Option, or if there's just a few messages being sent)

## thread replies 

Now you'll see an embed whenever someone is replying a thread with the content of the thread. This should add more context to messages whenever you see one on Discord.


# Discord Bot Configuration

You can look up on how to create a discord bot, but basically you'll need to go to the [Discord Developer Page](https://discord.com/developers/applications) and create a `New Application`. In the newly created application you can go to `Bot` and generate a new secret with the `Reset Token` button, which is basically your bot token.

Under OAuth -> URL Generator you gotta mark the option `bot`, and then `Administrator`. At the end of the page you will find a generated url that you have to copy and paste in a new tab. 

**Important:** make sure to invite this bot into an unused server. **Don't invite this bot into your main server**. The reason being that this bot manipulates Emojis, and it can mess up your server's existing emojis.

All you need now is the id of the server the bot is on, which you can get by enabling developer mode, right clicking your server, and clicking on `Copy Server ID`.

Both of these credentials (the token and the id) will be used in the `discordBot.yaml` file.
