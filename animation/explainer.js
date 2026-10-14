"use strict";

const scripts = {
  build: {
    duration: 18,
    steps: [
      [0, "The provider publishes AAA and its signature."],
      [0.12, "The approved workflow selects the exact base and builds BBB."],
      [0.43, "The build records the provider base digest in BBB's provenance."],
      [
        0.7,
        "The statement also identifies BBB and the workflow that built it.",
      ],
      [
        0.84,
        "Cosign signs the customer statement. The provider signature stays separate.",
      ],
    ],
  },
  verify: {
    duration: 20,
    steps: [
      [0, "The user supplies BBB's digest and the approved workflow commit."],
      [
        0.1,
        "Cosign retrieves BBB's signed provenance from the public registry.",
      ],
      [
        0.36,
        "Cosign verifies the customer statement. The base digest is now an authenticated claim.",
      ],
      [
        0.42,
        "The verifier uses that base reference to retrieve provider evidence separately.",
      ],
      [
        0.85,
        "Both signatures and the CUE policy pass. The build workflow remains a trust requirement.",
      ],
    ],
  },
};

const clamp = (value) => Math.min(1, Math.max(0, value));
const ease = (value) => 1 - Math.pow(1 - clamp(value), 3);
const reducedMotion = matchMedia("(prefers-reduced-motion: reduce)");
const scenes = [];

class Diagram {
  constructor(root) {
    this.root = root;
    this.config = scripts[root.dataset.scene];
    this.progress = 0;
    this.playing = false;
    this.visible = true;
    this.lastTime = null;
    this.frame = null;
    this.lastStep = -1;
    this.playButton = root.querySelector(".play-button");
    this.scrubber = root.querySelector('input[type="range"]');
    this.clock = root.querySelector(".clock");
    this.caption = root.querySelector(".scene-caption");
    this.counter = root.querySelector(".step-index");
    this.reveals = [...root.querySelectorAll("[data-after]")];
    this.checks = [...root.querySelectorAll("[data-check]")];
    this.packets = [...root.querySelectorAll("[data-route]")].map((node) => {
      const path = root.querySelector(`#${node.dataset.route}`);
      return {
        node,
        path,
        length: path.getTotalLength(),
        start: Number(node.dataset.start),
        end: Number(node.dataset.end),
      };
    });
    this.playButton.addEventListener("click", () => this.toggle());
    root.querySelector(".replay-button").addEventListener("click", () => {
      this.seek(0);
      this.setPlaying(true);
    });
    this.scrubber.addEventListener("input", () => {
      this.setPlaying(false);
      this.seek(Number(this.scrubber.value) / 1000);
    });
    new IntersectionObserver(
      ([entry]) => {
        this.visible = entry.isIntersecting;
        this.syncClock();
      },
      { threshold: 0.1 },
    ).observe(root);
    this.render();
  }

  toggle() {
    if (!this.playing && this.progress === 1) this.seek(0);
    this.setPlaying(!this.playing);
  }

  setPlaying(value) {
    this.playing = value;
    this.playButton.innerHTML = value
      ? "Ⅱ <span>Pause</span>"
      : "▶ <span>Play</span>";
    const kind = this.root.dataset.scene === "build" ? "build" : "verification";
    this.playButton.setAttribute(
      "aria-label",
      `${value ? "Pause" : "Play"} ${kind} animation`,
    );
    this.syncClock();
  }

  syncClock() {
    if (this.frame !== null) cancelAnimationFrame(this.frame);
    this.frame = null;
    this.lastTime = null;
    if (this.playing && this.visible && !document.hidden) {
      this.frame = requestAnimationFrame((time) => this.tick(time));
    }
  }

  tick(time) {
    if (this.lastTime !== null)
      this.progress = clamp(
        this.progress + (time - this.lastTime) / (1000 * this.config.duration),
      );
    this.lastTime = time;
    this.render();
    if (this.progress === 1) {
      this.setPlaying(false);
      return;
    }
    this.frame = requestAnimationFrame((next) => this.tick(next));
  }

  seek(progress) {
    this.progress = clamp(progress);
    this.lastTime = null;
    this.render();
  }

  render() {
    const progress = this.progress;
    this.root.dataset.progress = progress.toFixed(3);
    this.scrubber.value = Math.round(progress * 1000);
    const seconds = Math.floor(progress * this.config.duration)
      .toString()
      .padStart(2, "0");
    this.clock.textContent = `00:${seconds} / 00:${this.config.duration}`;
    let step = 0;
    this.config.steps.forEach(([start], index) => {
      if (progress >= start) step = index;
    });
    if (step !== this.lastStep) {
      this.caption.textContent = this.config.steps[step][1];
      this.counter.textContent = `${String(step + 1).padStart(2, "0")} / 05`;
      this.lastStep = step;
    }
    for (const node of this.reveals) {
      const amount = reducedMotion.matches
        ? Number(progress >= Number(node.dataset.after))
        : ease((progress - Number(node.dataset.after)) / 0.055);
      node.style.opacity = amount;
      node.style.transform = `translateY(${(1 - amount) * 9}px)`;
    }
    for (const { node, path, length, start, end } of this.packets) {
      const amount = (progress - start) / (end - start);
      const present = amount > 0 && amount < 1 && !reducedMotion.matches;
      node.style.opacity = present
        ? Math.min(amount * 12, (1 - amount) * 12, 1)
        : 0;
      const distance = node.hasAttribute("data-reverse")
        ? 1 - ease(amount)
        : ease(amount);
      const point = path.getPointAtLength(clamp(distance) * length);
      node.setAttribute("transform", `translate(${point.x} ${point.y})`);
    }
    for (const node of this.checks) {
      const done = progress >= Number(node.dataset.check);
      node.dataset.done = String(done);
      node.querySelector(".check-mark").textContent = done ? "✓" : "○";
    }
    if (this.root.dataset.scene === "build") {
      this.root.style.setProperty(
        "--build-progress",
        clamp((progress - 0.2) / 0.15),
      );
      this.root.querySelector(".build-state").textContent =
        progress < 0.2
          ? "Ready to build"
          : progress < 0.35
            ? "Build in progress"
            : "Build complete";
    }
  }
}

document
  .querySelectorAll("[data-scene]")
  .forEach((root) => scenes.push(new Diagram(root)));
document.addEventListener("visibilitychange", () =>
  scenes.forEach((scene) => scene.syncClock()),
);
reducedMotion.addEventListener("change", () =>
  scenes.forEach((scene) => {
    scene.setPlaying(false);
    scene.render();
  }),
);

const toggle = document.querySelector("#wrong-base");
const policy = document.querySelector(".policy-scene");
toggle.addEventListener("change", () => {
  const wrong = toggle.checked;
  policy.dataset.wrong = String(wrong);
  policy.querySelector(".expected-base").textContent = wrong
    ? "sha256:00000000…"
    : "sha256:e4b8f4cc…";
  policy.querySelector(".policy-verdict").innerHTML = wrong
    ? "<span>×</span> CUE: wrong base"
    : "<span>✓</span> CUE: matching base";
  policy.querySelector(".policy-result strong").textContent = wrong
    ? "Reject"
    : "Accept";
  policy.querySelector(".policy-result > span:last-child").textContent = wrong
    ? "The signature stays valid. The signed base fails the policy."
    : "The signed base matches the approved base.";
});
