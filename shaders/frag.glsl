#version 330 core

uniform int   uMode;
uniform vec4  uColor;
uniform vec4  uColor2;
uniform float uTime;
uniform vec3  uShapeCenter;
uniform float uShapeRadius;

uniform float uFogNear;
uniform float uFogFar;
uniform vec4  uFogColor;

in vec3 vCamPos;
out vec4 fragColor;

void main() {
    if (uMode == 0) {
        fragColor = uColor;
    } else if (uMode == 1) {
        float d = distance(vCamPos, uShapeCenter);
        float t = clamp(d / max(uShapeRadius, 1.0), 0.0, 1.0);
        fragColor = mix(uColor, uColor2, t);
    } else if (uMode == 2) {
        float brightness = 0.6 + 0.4 * sin(uTime * 3.0);
        fragColor = vec4(uColor.rgb * brightness, uColor.a);
    } else {
        fragColor = uColor;
    }

    float depth = -vCamPos.z;
    float fog_t = clamp((depth - uFogNear) / (uFogFar - uFogNear), 0.0, 1.0);
    fragColor = mix(fragColor, uFogColor, fog_t);
}
