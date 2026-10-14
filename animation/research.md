# Research on animated technical explainers

Research date: 14 October 2026.

The first Manim draft used the same composition for each chapter and mostly faded text boxes.
That explained the sequence, but it did not establish a strong visual style or use motion to explain relationships.

## The supplied Anomaly reference

[OpenCode 2.0](https://anoma.ly/notes/opencode-2-0/) contains live diagrams with playback controls.
I inspected the page in a browser and examined its public JavaScript and CSS assets.

The served code contains React, Motion/Framer Motion, SVG, shared clocks, viewport observation, and reduced-motion handling.
The service diagram keeps its clients and service in fixed locations.
Connections, messages, session rows, and status changes show the operation of the system.
The API diagram moves a session between worktrees instead of replacing the whole diagram.

The visual design uses small monospaced labels, thin connections, restrained color, and consistent spacing.
Prose explains the concept beside the diagrams. The diagrams show the mechanism.
The page includes video too, but its technical diagrams are browser components.

These observations establish the implementation approach. They do not establish which coding agent, if any, created the page.
The public bundles contain much more than these diagrams. A small explanation does not need the entire application stack.

## Useful public projects

### Video Shotcraft

[Repository and examples](https://github.com/Vincentwei1021/video-shotcraft) · [Motion gallery](https://vincentwei1021.github.io/video-shotcraft/)

This project supplies shot recipes, preview animations, a Remotion template, and reusable camera and typography components.
Its examples include progressive assembly, exploded views, and camera movement across interfaces.
The project documents specific timing and easing values rather than relying on a general request for smooth animation.
It also checks typography, reading time, camera stability, and excessive effects.

Useful lesson: choose a motion technique for a specific explanation, then adapt its tested timing.
Do not add every available effect.

### Sub-Level marketing videos

[Repository](https://github.com/sub-level/marketing-videos) · [Production process](https://github.com/sub-level/marketing-videos/blob/main/docs/process.md)

This project publishes source, case studies, and links to launch films.
Its method starts with one message, a storyboard, and a fixed visual specification.
It measures reference films for text size, arrival timing, hold duration, and transitions.
Its compositions preserve geometry between shots and use a common timing model for captions and sound.
The product demonstrations can use real application components or recordings on a 3D device.

Useful lesson: the objects should carry the explanation through the sequence.
Replacing every object between scenes produces a slideshow, even with good easing.
Some showcase films use paid footage and music providers. Those services are not necessary for our diagrams.

### Claude Motion

[Repository](https://github.com/JulianVolodia/claudestory_claude-motion)

This toolkit combines motion rules, a timeline, sound synthesis, and frame review.
It also has a browser simulation pipeline with a deterministic `render(t)` function.
That approach can produce a live explanation and allow frame-by-frame video capture from the same source.

Useful lesson: derive the picture from time, rather than chaining unrelated timers.
This makes playback, pause, scrubbing, review, and repeatable capture possible.
Its sound engine demonstrates that local sound generation can avoid another hosted service.

### Agentic Product Demo

[Repository](https://github.com/Alexwtlf/agentic-product-demo)

This project implements product interfaces as React compositions.
It defines named timing points for actions, pointer movement, counters, and row transitions.
It renders stills at important timing points and checks the finished frames.
Its documentation distinguishes automated frame checks from human visual review.

Useful lesson: a successful render does not establish good composition or pacing.
Inspect the actual output at the start, during transitions, and at each final state.

## Cost and tool choice

Animo is optional. It packages a renderer; it is not necessary to write or render Manim scripts.

[Remotion's license](https://github.com/remotion-dev/remotion/blob/main/LICENSE.md) has eligibility conditions for free use.
Local rendering does not remove those conditions. Do not assume every company can use it without a paid license.

Manim Community uses the MIT license and supports local video rendering.
Browser diagrams can also run locally with HTML, CSS, SVG, and JavaScript, without a rendering service.
Motion, GSAP, and Three.js can help larger projects. Our first browser explanation does not need these dependencies.

## Direction for this demo

Use an interactive article with three diagrams:

1. Show the provider base, customer build, and customer image in fixed locations.
2. Move the base digest into the customer statement. Keep the provider signature separate.
3. Follow the third-party verifier's requests to each public registry.
4. Keep the signed statement fixed while a policy change makes CUE reject it.

Use one visual system for all diagrams. Keep motion limited to changes that explain the process.
Use a deterministic timeline for the moving packets, revealed fields, status changes, and captions.
Provide keyboard controls, reduced-motion behavior, and a layout for narrow screens.
Keep real cryptographic verification in the existing script. Clearly label the browser explanation as an illustration.

The browser version is `explainer.html`, with local CSS and JavaScript.
It uses no CDN, account, rendering service, or extra package installation.
It takes techniques from the research. It does not copy another project's code, artwork, or branding.
