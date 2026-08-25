#include "lib_core.h"
#include "lib3d.h"
#include "libbody.h"
#include <math.h>
#include <stdlib.h>
#include <stdio.h>

#define CONE_LINES 20

int main(void) {
    if (!render_init(1600, 1000)) return 1;

    render_set_fog(100.0f, 600.0f, 0.25f, 0.25f, 0.25f, 1.0f);

    /* --- Szene aufbauen --- */
    Body *bodies = arr_create(0, sizeof(Body));

    Solid *box_mesh = solid_box(100, 80, 60); //Eine Box im GPU Speicher
    Solid *pyramid_mesh = solid_pyramid(90, 120); //Eine Pyramide im GPU Speicher
    Solid *grid_mesh = solid_grid(600, 24);

    Body grid = body_create(grid_mesh, 0, 0, 0, (BodyConfig){ .color = "#777774", .line_width = 1.0f });
    bodies = arr_push(bodies, &grid);

    Body box1 = body_create(box_mesh, 150, 0, 50, (BodyConfig){ .color = "#ff0000", .line_width = 2.0f });
    bodies = arr_push(bodies, &box1);

    Body box2 = body_create(box_mesh, 0, 0, 100, (BodyConfig){ .color = "#00ffff", .line_width = 2.0f });
    bodies = arr_push(bodies, &box2);

    Body box3 = body_create(box_mesh, -150, 0, -100, (BodyConfig){ .color = "#ff0000", .line_width = 2.0f });
    bodies = arr_push(bodies, &box3);

    Body box4 = body_create(box_mesh, -200, 0, 30, (BodyConfig){ .color = "#00ffff", .line_width = 2.0f, .rot_y = M_PI / 2.0f });

    bodies = arr_push(bodies, &box4);

    Body pyr1 = body_create(pyramid_mesh, 100, 0, -100, BODY_CONFIG_DEFAULT);
    bodies = arr_push(bodies, &pyr1);

    Line **lines = arr_create(0, sizeof(Line*));

    Vec3 apex = vec3(0, 0, 0);
    float cone_len = 600;
    float cone_angle = 3.14159265f / 60.0f;
    float cone_r = cone_len * sinf(cone_angle);
    float cone_z = cone_len * cosf(cone_angle);

    for (int i = 0; i < CONE_LINES; i++) {
        float a = (2.0f * 3.14159265f * i) / CONE_LINES;
        Vec3 end = vec3(
            apex.x + cosf(a) * cone_r,
            apex.y + sinf(a) * cone_r,
            apex.z + cone_z
        );
        Line *l = line_create(apex, end, "#ff8800", 1);
        lines = arr_push(lines, &l);
    }

    float time_accum = 0;
    float cone_rot_y = 0;
    int   frame_count = 0;
    float fps_last    = 0;

    /* --- Render-Loop --- */
    while (!render_should_close()) {
        render_frame_begin();

        time_accum += 0.02f;
        frame_count++;
        float now = (float)glfwGetTime();
        if (now - fps_last >= 2.0f) {
            float fps = frame_count / (now - fps_last);
            printf("FPS: %.1f\n", fps);
            frame_count = 0;
            fps_last   = now;
        }

        render_background(40, 40, 40);

        float cam_angle = time_accum * 0.15f;
        float cam_radius = sqrtf(40.0f * 40.0f + 180.0f * 180.0f);
        float cam_height = 140.0f;
        Vec3 cam_pos = vec3(
            sinf(cam_angle) * cam_radius,
            cam_height,
            cosf(cam_angle) * cam_radius
        );
        Vec3 target = vec3(0, 0, 0);
        Vec3 up     = vec3(0, 1, 0);

        Mat4x4 view = mat4x4_lookat(cam_pos, target, up);
        Mat4x4 proj = mat4x4_perspective(1.2f, render_get_aspect(), 0.1f, 1000.0f);
        render_set_projection(&proj);

        size_t body_count = arr_len(bodies);
        for (size_t i = 0; i < body_count; i++)
            body_draw(&bodies[i], &view);

        cone_rot_y += 0.01f;
        Mat4x4 cone_rot = mat4x4_rotate(0, cone_rot_y, 0);

        size_t line_count = arr_len(lines);
        PlaneArray *box_planes = arr_create(body_count, sizeof(PlaneArray));
        for (size_t i = 0; i < body_count; i++) {
            Body *b = &bodies[i];
            int vc = b->solid->vertex_count;
            Vec3 *world_verts = malloc((size_t)vc * sizeof(Vec3));
            Mat4x4 rot = mat4x4_rotate(b->rot_x, b->rot_y, b->rot_z);
            for (int j = 0; j < vc; j++)
                world_verts[j] = vec3_add(vec3_transform(b->solid->vertices[j], &rot), b->pos);
            box_planes[i] = body_get_face_planes(b, world_verts);
            free(world_verts);
        }

        for (size_t i = 0; i < line_count; i++) {
            Vec3 rotated_end = rotate_around(lines[i]->p2, lines[i]->p1, &cone_rot);
            Vec3 endpoint = rotated_end;
            float max_dist = vec3_squared_length(vec3_sub(rotated_end, lines[i]->p1));

            for (size_t j = 0; j < body_count; j++) {
                for (int k = 0; k < box_planes[j].count; k++) {
                    Vec3 hit;
                    if (plane_intersect_line(&box_planes[j].data[k], lines[i]->p1, rotated_end, &hit)) {
                        float dist = vec3_squared_length(vec3_sub(hit, lines[i]->p1));
                        if (dist < max_dist) {
                            endpoint = hit;
                            max_dist = dist;
                        }
                    }
                }
            }

            line_draw(lines[i], &view, &endpoint);
            render_stroke_color_hex("#ff0000");
            render_point_size(5);
            render_point(endpoint.x, endpoint.y, endpoint.z);
        }

        for (size_t i = 0; i < body_count; i++)
            plane_array_free(&box_planes[i]);
        arr_free(box_planes);

        render_frame_end();
    }

    /* --- Aufraumen --- */
    for (size_t i = 0; i < arr_len(bodies); i++)
        body_destroy(&bodies[i]);
    arr_free(bodies);

    for (size_t i = 0; i < arr_len(lines); i++)
        line_free(lines[i]);
    arr_free(lines);

    glfwTerminate();
    return 0;
}
