// Ghostty Studio · Soft glow
// Copyright (c) 2026 Daniel Gosek. MIT License; see ../../LICENSE.
// A restrained, static highlight bloom. Five texture samples per pixel.
void mainImage(out vec4 fragColor, in vec2 fragCoord) {
    vec2 uv = fragCoord / iResolution.xy;
    vec2 px = 1.25 / iResolution.xy;
    vec4 original = texture(iChannel0, uv);
    vec3 neighbours = (
        texture(iChannel0, uv + vec2(px.x, 0.0)).rgb +
        texture(iChannel0, uv - vec2(px.x, 0.0)).rgb +
        texture(iChannel0, uv + vec2(0.0, px.y)).rgb +
        texture(iChannel0, uv - vec2(0.0, px.y)).rgb) * 0.25;
    vec3 halo = max(neighbours - vec3(0.25), vec3(0.0));
    fragColor = vec4(min(original.rgb + halo * 0.10, vec3(1.0)), original.a);
}
