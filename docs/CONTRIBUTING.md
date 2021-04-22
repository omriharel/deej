I'm writing this document to help set the expectations of developers looking to add features they wrote to deej's main fork.

I hope that even though not everyone would be happy about my approach to this subject, we'll be able to have a civil and respectful discussion around it. If you're interested in doing so, I invite you to reach out to me via our community [Discord server](https://discord.gg/nf88NJu).

Thanks for reading!

## Some background, and my vision for deej

Similarly to other software developers who maintain open-source projects outside of their workplace, I work on deej - for free - in my free time, motivation and mood permitting.

A lot of that time is spent interacting outside of GitHub, by running our community Discord server which is an active space with over a thousand members. This includes supporting users with their initial setup of deej, answering questions about deej and commenting on user-created builds and designs.

Since the project's initial debut in February 2020, I had the pleasure of talking to hundreds of users - beginners and seasoned developers alike - which has guided my decisions on subjects like licensing and documentation, as well as helped me form the following vision for deej's future:

### Project scope

I have a fairly set vision for deej and what it should be (including at which point in time I'd like to add certain things). I prefer to work on these things myself - as mentioned above, when I have the time and motivation to do so.

### Project audience

Many of deej's users aren't necessarily tech-savvy, and for some of them this is their first time making a combined electronics hardware + software project. This fact influences many decisions vis-a-vis keeping things as simple as possible. I care a lot about beginners being able to get started easily, even at the cost of certain more advanced features not being included in vanilla deej.

## Pull requests and alternate forks

The nature of how I currently choose to maintain deej means that **I'm not likely to accept and incorporate PRs** into deej's main fork ([omriharel/deej](https://github.com/omriharel/deej)).

Despite the above, **deej is still a fully open-source project**. I don't want my occasional lack of energy to stand in the way of anyone looking to make something awesome - you have my blessing to fork the project, maintain your copy of it separately, and tell the world (_including_ our community Discord server) about it!

### Getting started with development

- Have a Go 1.14+ environment
- Use the build scripts under `pkg/deej/scripts` for your built binaries if you want them to have the notion of versioning

## Issues

I welcome all bug reports and feature requests, and try to respond to these within a reasonable amount of time.
