// Ghostty Studio · Soft scanlines
// Copyright (c) 2026 Daniel Gosek. MIT License; see ../../LICENSE.
// A faint static raster, without warp, flicker, or chromatic aberration.
void mainImage(out vec4 fragColor, in vec2 fragCoord) {
    vec4 original = texture(iChannel0, fragCoord / iResolution.xy);
    float line = 0.975 + 0.025 * cos(fragCoord.y * 3.14159265);
    fragColor = vec4(original.rgb * line, original.a);
}
