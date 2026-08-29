# Hello

And this website is a working demo of "backend-for-frontend" (pun intended) architecture.
You are looking at a binary that renders text-based user interface 
to an actual Linux shell running in a container and piping bytes to your browser via WebSockets, 
all from a k3s single-node cluster on a VPS.
Yes, that's a mouthful, in other words:

## Hi, I'm Andy

I live and write code in Berlin (... or is it generating code now?). 
I also sort of paint, but that's another story. 
I have about 10 years of production experience with web and cloud projects of 
different shapes and sizes: from learning platforms to insurance systems to
giants of e-commerce. Ruby and Rails are my bread and butter but I also dabble
in Go, JavaScript or anything else I want to have fun with. It goes without saying
that I also write a lot of YAML. 

I came to this from international news broadcasting — I was a reporter, foreign
correspondent, and European bureau chief for 13 years before I retrained myself as a developer. 
I have never fully severed the writing habit, which is how I ended up running the technical blog at Evil Martians for five years.

Currently I'ma a senior backend engineer at Sofatutor, working
on payment and subscription systems — the kind of code where being off by couple of cents or seconds
cent is at least very disappointed customer and a blow to the revenue at most.

Right before: Shopify (production engineering) andv Getsafe (backend).

[Email](mailto:andrey@hey.com)
[Github](https://github.com/progapandist)
[Site](https://progapanda.org)

# Experience

## Where I've worked

- Senior Backend Engineer, Sofatutor — Sep 2025 to present
  Building and maintaining payment and subscription systems for a German
  online learning platform. Billing logic, recurring charges, and the
  unglamorous correctness work that money demands.

- Career pause — Jun 2024 to Sep 2025
  A deliberate break after several intense years at Shopify and Getsafe. Also
  the reason for the lull on my Github.

- Senior Backend Engineer, Getsafe — Dec 2023 to May 2024
  German digital insurance scale-up. Led an internal R&D effort to modernize
  the Policy Administration System, a complex legacy Rails codebase, and
  delivered an actionable refactoring blueprint for mission-critical insurance
  infrastructure.

- Senior Production Engineer, Shopify — Sep 2021 to Oct 2023
  Spearheaded a new internal Rails monolith replacing SlackOps with a secure
  infra portal. Designed and maintained lifecycle tooling for the Kubernetes
  clusters powering internal services. Focused on self-service platform design,
  developer experience, and secure automation.

- Full-Stack Developer / Rails Instructor, Le Wagon — 2019 to 2021
  Co-developed the learning management platform for the leading in-person
  programming bootcamp. Built and containerized grading tools for student code,
  ran the cloud infrastructure, acted as SRE for multiple services, and taught
  Ruby and Rails to career switchers across Europe.

- Head of Content, Evil Martians — 2017 to 2022
  Ran editorial strategy for Martian Chronicles. Wrote and edited technical
  articles that landed on Hacker News, occasionally at #1.

- Freelance Rails Developer — 2017 to 2018
  Business-logic-heavy Rails applications for European startups.

# Skills

## What I work with

- Daily drivers
  Ruby, Rails, JavaScript, TypeScript, SQL, Redis

- Infrastructure
  Kubernetes, Docker, GCP, Digital Ocean, GitHub Actions, Buildkite

- Also in the toolbox
  Go (basics — this program included), Python, Java, Swift, C

- The parts that aren't a language
  CI/CD pipelines, containerized environments, systems thinking, documentation
  that people actually read, and mentoring.

# Open Source

## Things I've put out there

- [stripeek](https://github.com/progapandist/stripeek)
  A debugging tool for Stripe — for when you need to see what the API is
  actually doing, not what the dashboard says it did. Quit this program and
  type `stripeek` for an interactive demo.

- [tja](https://github.com/progapandist/tja)
  German prefix verbs as a one-armed bandit: two reels, prefixes and stems,
  that filter each other. Quit this program and type `tja` — it works with a
  mouse, and on a phone by tapping.

- [rails-k8s-demo](https://github.com/lewagon/rails-k8s-demo)
  A complete worked example for deploying Sidekiq-backed Rails apps to
  DigitalOcean Kubernetes.

- [foot_traffic](https://github.com/lewagon/foot_traffic)
  Pure Ruby DSL for headless Chrome scripting via Ferrum. No Selenium.

- [wait-on-check-action](https://github.com/lewagon/wait-on-check-action)
  GitHub Action that halts a workflow until required checks pass on a ref.

- [progapanda.org](https://github.com/progapandist/progapanda.org)
  This website. An experimental terminal UI in Go, piped over WebSockets.

[Github](https://github.com/progapandist)

# Background

## The long way round

- Education
  Computer Science (unfinished BSc), Vrije Universiteit Amsterdam, 2017-2019.
  Full Stack Web Development Bootcamp, Le Wagon Paris, 2016.
  Master's in International Journalism, Lomonosov Moscow State University,
  2004-2009.

- Before software
  Started at 16 as a tech writer at Yandex in 2001. Then 13 years in broadcast
  journalism — editor, reporter, foreign correspondent, European bureau chief.
  I left the profession in 2014, after the annexation of Crimea, and retrained
  into software development.

- Where
  Brussels, Paris, Amsterdam, Berlin — moving around since 2010.

- Languages
  English (fluent), French (fluent), Russian (native), German (basic
  conversational).

- Paperwork
  Permanent German resident, unrestricted work rights.

# How

## How it's built

- An Xterm.js front-end emulates a terminal in your browser.

- A Go server upgrades the connection to a WebSocket.

- An Alpine container starts on the backend, one per visitor, with no network
  and a hard cap on CPU and memory.

- A Go binary inside that container renders this TUI. The original was written
  in 2020 with tview; this rewrite uses Bubble Tea, Lip Gloss, and Bubbles.

- Stdin and stdout of the container are piped back and forth over the
  WebSocket.

The whole thing is open source:

[Source](https://github.com/progapandist/progapanda.org)

# Quit

## Thanks for stopping by

Press Enter (or q) to close this session.

If you want to talk about anything work-related, don't hesitate 
to contact me by email.

[Email](mailto:andrey@hey.com)
