// Ghostty Studio · Paper grain
// Copyright (c) 2026 Daniel Gosek. MIT License; see ../../LICENSE.
// Stationary, monochrome grain. No animation or color shift.
void mainImage(out vec4 fragColor, in vec2 fragCoord) {
    vec4 original = texture(iChannel0, fragCoord / iResolution.xy);
    float grain = fract(sin(dot(floor(fragCoord), vec2(12.9898, 78.233))) * 43758.5453);
    float shade = 1.0 - grain * 0.025;
    fragColor = vec4(original.rgb * shade, original.a);
}
