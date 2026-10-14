"""Render the signed-base explanation with Manim Community."""

from manim import (
    BLUE_C, DOWN, FadeIn, FadeOut, GREEN_C, GrowArrow, Indicate, LEFT,
    ORANGE, RIGHT, RoundedRectangle, Scene, Text, Transform,
    TransformFromCopy, UP, VGroup, WHITE, Arrow,
)


BASE = "sha256:e4b8f4cc..."
RUNTIME = "sha256:07247ee0..."
CUSTOMER = "sha256:d0a59cb7..."
BACKGROUND = "#101923"


class BaseProvenance(Scene):
    """Explain the existing public demo with abbreviated, real digests."""

    def text(self, value, size=25, color=WHITE):
        return Text(value, font="DejaVu Sans", font_size=size, color=color)

    def panel(self, title, lines, color=BLUE_C, width=4.8, height=2.2):
        frame = RoundedRectangle(
            width=width, height=height, corner_radius=0.16,
            stroke_color=color, fill_color=BACKGROUND, fill_opacity=1,
        )
        content = VGroup(
            self.text(title, 29, color),
            *[self.text(line, 22) for line in lines],
        ).arrange(DOWN, buff=0.22)
        if content.width > width - 0.4:
            content.scale_to_fit_width(width - 0.4)
        return VGroup(frame, content)

    def chapter(self, number, title, caption):
        if self.mobjects:
            self.play(*[FadeOut(item) for item in list(self.mobjects)], run_time=0.5)
        heading = self.text(f"{number}. {title}", 38).to_edge(UP, buff=0.5)
        footer = self.text(caption, 25).to_edge(DOWN, buff=0.6)
        if footer.width > 13:
            footer.scale_to_fit_width(13)
        self.play(FadeIn(heading), FadeIn(footer), run_time=0.7)

    def connect(self, start, end, color=WHITE):
        arrow = Arrow(start, end, buff=0.14, color=color, stroke_width=3)
        self.play(GrowArrow(arrow), run_time=0.7)
        return arrow

    def construct(self):
        self.camera.background_color = BACKGROUND

        self.chapter(1, "The provider signs the base", "The provider signature belongs to image AAA.")
        base = self.panel("AAA: Red Hat core-runtime", [BASE, "Multi-platform image index"]).shift(LEFT * 3.4)
        signature = self.panel("Provider signature", ["Verified with the Red Hat public key"], GREEN_C, 5, 1.4).shift(RIGHT * 3.4)
        self.play(FadeIn(base), run_time=0.8)
        self.connect(base.get_right(), signature.get_left(), GREEN_C)
        self.play(FadeIn(signature), run_time=0.8)
        self.wait(4)

        self.chapter(2, "The customer builds a new image", "The base signature stays separate from the customer image.")
        base = self.panel("AAA: signed base", [BASE]).shift(LEFT * 3.4 + DOWN * 0.1)
        customer = self.panel("BBB: customer image", [CUSTOMER, "Base layers + customer file"], ORANGE).shift(RIGHT * 3.4 + DOWN * 0.1)
        signature = self.text("Provider signature", 23, GREEN_C).next_to(base, DOWN, buff=0.4)
        file = self.text("+ /customer/message.txt", 24, ORANGE).move_to(UP * 1.8)
        self.play(FadeIn(base), FadeIn(signature))
        self.connect(base.get_right(), customer.get_left())
        self.play(TransformFromCopy(base[0], customer[0]), FadeIn(customer[1]), run_time=1)
        self.play(FadeIn(file), run_time=0.6)
        self.wait(4)

        self.chapter(3, "The build records its inputs", "Provenance records digests. It does not embed the provider signature.")
        statement = self.panel("SLSA provenance for BBB", [
            f"Output image: {CUSTOMER}",
            f"Provider base: {BASE}",
            f"AMD64 base: {RUNTIME}",
            "Workflow identity + source commit",
        ], BLUE_C, 8.9, 3.5)
        self.play(FadeIn(statement), run_time=0.8)
        self.play(Indicate(statement[1][2], color=GREEN_C), run_time=1)
        self.wait(5)
        seal = self.text("Cosign signs this statement with the GitHub workflow identity", 25, GREEN_C).shift(DOWN * 2.3)
        self.play(FadeIn(seal))
        self.wait(4)

        self.chapter(4, "A third party checks both signatures", "Public evidence allows verification without a registry login.")
        customer = self.panel("Customer statement", ["Cosign checks signature", "Output digest + approved workflow"], ORANGE, 5.2, 2.3).shift(LEFT * 3.4 + UP * 0.3)
        provider = self.panel("Provider evidence", ["Cosign checks base signature", "Uses trusted provider keys"], GREEN_C, 5.2, 2.3).shift(RIGHT * 3.4 + UP * 0.3)
        self.play(FadeIn(customer), run_time=0.8)
        reference = self.text(BASE, 23, BLUE_C).shift(DOWN * 1.8 + LEFT * 3)
        self.play(FadeIn(reference))
        self.play(reference.animate.move_to(DOWN * 1.8 + RIGHT * 3), run_time=2)
        self.play(FadeIn(provider), run_time=0.8)
        link = self.connect(customer.get_right(), provider.get_left(), BLUE_C)
        self.play(Indicate(link), run_time=1)
        self.wait(5)

        self.chapter(5, "CUE checks the acceptance rules", "A valid signature can cover a statement about the wrong base.")
        recorded = self.panel("Signed statement", [f"Recorded base: {BASE}", "Signature: valid"], BLUE_C, 5.2, 2.3).shift(LEFT * 3.4 + UP * 0.4)
        expected = self.panel("Your CUE policy", [f"Required base: {BASE}", "Other required build inputs"], GREEN_C, 5.2, 2.3).shift(RIGHT * 3.4 + UP * 0.4)
        result = self.text("MATCH: accept the statement", 30, GREEN_C).shift(DOWN * 1.8)
        self.play(FadeIn(recorded), FadeIn(expected))
        self.connect(recorded.get_right(), expected.get_left())
        self.play(FadeIn(result))
        self.wait(3)
        wrong = self.panel("Your CUE policy", ["Required base: sha256:00000000...", "Other required build inputs"], ORANGE, 5.2, 2.3).move_to(expected)
        rejected = self.text("MISMATCH: reject, although the signature is valid", 29, ORANGE).move_to(result)
        self.play(Transform(expected, wrong), Transform(result, rejected), run_time=1.3)
        self.wait(5)

        self.chapter(6, "What must users trust?", "An approved builder can lie. Signatures cannot make false claims true.")
        trust = VGroup(
            self.text("Provider public keys", 32, GREEN_C),
            self.text("Customer build workflow and approved commit", 32, BLUE_C),
            self.text("Build environment", 32, BLUE_C),
        ).arrange(DOWN, buff=0.6).shift(UP * 0.2)
        self.play(FadeIn(trust), run_time=1)
        self.wait(5)
        limit = self.text("Base origin does not prove unchanged files or preserved hardening.", 24).shift(DOWN * 2)
        self.play(FadeIn(limit))
        self.wait(4)
