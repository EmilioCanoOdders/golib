#version 330

// CRT: a whisper of a tube screen, adapted from games/asteroids' crt.fs with
// the flicker removed, the scanlines softened and the dark border much
// lighter, so the effect stays subtle. The colors split a screen pixel at the
// edges, and the picture bulges as little as the game's curvature uniform
// asks.

in vec2 fragTexCoord;
in vec4 fragColor;

uniform sampler2D texture0;
uniform vec2 screenSize; // set by GoLib: the game's screen, in pixels
uniform vec2 outputSize; // set by GoLib: the picture on the window, in pixels
uniform float curvature; // set by the game: 0 is flat

out vec4 finalColor;

// curve bends texture coordinates outwards from the center, like a tube.
vec2 curve(vec2 uv)
{
    uv = uv * 2.0 - 1.0;
    vec2 offset = abs(uv.yx) * curvature;
    uv = uv + uv * offset * offset;
    return uv * 0.5 + 0.5;
}

void main()
{
    vec2 uv = curve(fragTexCoord);
    if (uv.x < 0.0 || uv.x > 1.0 || uv.y < 0.0 || uv.y > 1.0) {
        finalColor = vec4(0.0, 0.0, 0.0, 1.0);
        return;
    }

    // Red and blue read one screen pixel apart.
    vec2 shift = vec2(1.0 / screenSize.x, 0.0);
    vec3 color = vec3(
        texture(texture0, uv + shift).r,
        texture(texture0, uv).g,
        texture(texture0, uv - shift).b);

    // A scanline every 3 pixels of the window, so they stay sharp at any
    // size, a breath darker than the picture: the pale fog would show any
    // deeper one as stripes.
    float scanline = sin(uv.y * outputSize.y * 3.14159265 / 1.5) * 0.5 + 0.5;
    color *= mix(0.96, 1.0, scanline);

    // A touch darker towards the corners.
    vec2 centered = uv - 0.5;
    color *= 1.0 - dot(centered, centered) * 0.3;

    finalColor = vec4(color, 1.0);
}
