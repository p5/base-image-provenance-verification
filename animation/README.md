# Render the explanation locally

## Interactive browser explanation

Open `animation/explainer.html` in a browser. No server or package installation is necessary.
Play each diagram, pause it, or move its progress slider.
Select **Require a different base** to see CUE reject a statement whose signature stays valid.

This version follows the live-diagram approach in the supplied Anomaly reference.
See [the research](research.md) for the inspected examples, production methods, and cost constraints.
The browser explanation is an illustration. It does not perform cryptographic verification.

## Original Manim video

Manim Community is free software under the MIT license.
Rendering uses your computer. It does not require Animo, a subscription, or a hosted rendering service.

The video explains the existing public demo. It uses abbreviated real image digests.
It does not perform verification. Use `scripts/verify.sh` for verification.
The video has on-screen captions and no audio.

## Render with Podman

Install Podman, `curl`, and `tar`. These commands support a Linux x86_64 host.
Run them from the repository root:

```bash
mkdir -p results/animation
curl -fL https://github.com/astral-sh/uv/releases/download/0.13.1/uv-x86_64-unknown-linux-musl.tar.gz \
  -o results/animation/uv.tar.gz
tar -xzf results/animation/uv.tar.gz -C results/animation
podman run --rm --userns=keep-id --user "$(id -u):$(id -g)" \
  --env HOME=/tmp --env UV_CACHE_DIR=/tmp/uv-cache \
  --volume "$PWD:/work:Z" \
  --workdir /work --entrypoint /bin/bash \
  docker.io/manimcommunity/manim:v0.19.0 \
  -c '/work/results/animation/uv-x86_64-unknown-linux-musl/uv venv --system-site-packages /tmp/manim-venv && /work/results/animation/uv-x86_64-unknown-linux-musl/uv run --no-project --python /tmp/manim-venv/bin/python -m manim -qm --media_dir results/animation animation/provenance.py BaseProvenance'
```

The first run downloads the container image. Subsequent renders use the local image.
The command creates a temporary Python virtual environment inside the container.
It uses the container's installed Manim dependencies. It does not change your system Python.
The downloaded `uv` executable is a static Linux binary. It also works inside the container.

Open `results/animation/videos/provenance/720p30/BaseProvenance.mp4` in your video player.
Edit `animation/provenance.py`, then repeat the render command.
Replace `-qm` with `-ql` for a faster draft or `-qh` for 1080p output.

## Render with an existing local Manim environment

If your system already has Cairo and Pango development libraries, use an isolated Python environment:

```bash
uv venv --python 3.13 results/manim-venv
uv pip install --python results/manim-venv/bin/python 'manim==0.19.0'
uv run --no-project --python results/manim-venv/bin/python -m manim \
  -qm --media_dir results/animation animation/provenance.py BaseProvenance
```

This script uses plain text. It does not require LaTeX.

## How people use coding agents for animation

1. Describe the concept and the facts that must stay accurate.
2. Ask the agent to write a short storyboard.
3. Ask the agent to implement each scene in Manim or Remotion.
4. Render a draft locally.
5. Review the frames and captions. Ask for specific changes.
6. Render the final MP4 and keep the source code.

Manim uses Python and suits technical explanations with moving diagrams.
Remotion uses React and suits product videos with interface elements, audio, and video assets.
Both let a coding agent edit animation source code.
For this explanation, Manim needs no paid rendering service.

References:

- [Animo's description of the agent-to-Manim workflow](https://animo.video/opencode-manim).
- [Manim quickstart](https://docs.manim.community/en/stable/tutorials/quickstart.html).
- [Manim container instructions](https://docs.manim.community/en/stable/installation/docker.html).
- [Manim MIT license](https://github.com/ManimCommunity/manim/blob/main/LICENSE).
- [Remotion's skills for Codex and Claude Code](https://www.remotion.dev/docs/ai/skills).
