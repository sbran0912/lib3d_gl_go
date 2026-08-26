#include "lib_core.h"
#include "lib3d.h"
#include "libbody.h"
#include <stdbool.h>

typedef struct {
    Body     body;
    Vec3     vel;
    Vec3     accel;
    Vec3     heading;
    float health;
    float dna[4];
} Vehicle;

Vehicle vehic_create(Body body) {
    Vehicle result = {
        .body         = body,
        .vel          = vec3(0, 0, 0),
        .accel        = vec3(0, 0, 0),
        .heading      = vec3(0, 0, 0),
        .health = 1,
        .dna[0] = random_float(-1.0, 1.0),   //Force to Poison
        .dna[1] = random_float(-1.0, 1.0),   //Force to good Food
        .dna[2] = random_float(20.0, 60.0), //Radius to Poison
        .dna[3] = random_float(20.0, 60.0)  //Radius to good Food

    };
    return result;
}

void vehic_destroy(Vehicle *v) {
    body_destroy(&v->body);
}

void vehic_alignToVelocity(Vehicle *vehic) {
    Vec3 vel = vehic->vel;

    float mag = sqrt(vel.x * vel.x + vel.y * vel.y + vel.z * vel.z);
    if (mag < 0.0001) return;

    float magXZ = sqrt(vel.x * vel.x + vel.z * vel.z);

    vehic->body.rot_x = acos(vel.y / mag);
    vehic->body.rot_y = (magXZ < 0.0001) ? 0.0f : atan2(vel.x / magXZ, vel.z / magXZ);
    vehic->body.rot_z = 0;

    /* heading als normalisierte Richtung ableiten */
    vehic->heading = vec3(vel.x / mag, vel.y / mag, vel.z / mag);
}

void vehic_applyForce(Vehicle *vehic, Vec3 force) {
    vehic->accel = vec3_add(vehic->accel, force);
}

void vehic_update(Vehicle* vehic) {
    //printf("accel: %f \n", vec3_length(vehic->accel));
    vehic->vel = vec3_add(vehic->vel, vehic->accel);
    float speed = vec3_length(vehic->vel);
    vehic->vel = vec3_scale(vec3_normalize(vehic->vel), constrain_num(speed, 0.5f, 2.0f));

    //printf("vel: %f \n", vec3_length(vehic->vel));
    vehic->accel = (Vec3){0.0f, 0.0f, 0.0f};
    vehic->body.pos = vec3_add(vehic->body.pos, vehic->vel);
}

void vehic_seek(Vehicle* vehic, Vec3 target, bool is_badfood) {
    Vec3 desired = vec3_limit(vec3_sub(target, vehic->body.pos), 3.0f);
    if (is_badfood) {
        desired = vec3_scale(desired, vehic->dna[0]);
    } else {
        desired = vec3_scale(desired, vehic->dna[1]);
    }

    Vec3 steer = vec3_limit(vec3_sub(desired, vehic->vel), 2.0f);
    vehic_applyForce(vehic, vec3_scale(steer, 0.2));
}

Body* food_create(size_t count, const char *color, Solid* single_food_mesh) {
    Body* food = arr_create(count, sizeof(Body));
    for (size_t i = 0; i < count; i++) {
        Body single_food = body_create(single_food_mesh,
                                       random_int(-100, 100),
                                       random_int(-100, 100),
                                       random_int(-100, 100),
                                       (BodyConfig){ .color = color, .line_width = 1.0f });
        food = arr_push(food, &single_food);
    }
    return food;
}

Body* food_respawn(Body* food, Solid* single_food_mesh, size_t min, size_t count, const char *color) {
    if (arr_len(food) < min) {
        for (size_t i = 0; i < count; i++) {
            Body single_food = body_create(single_food_mesh,
                                           random_int(-100, 100),
                                           random_int(-100, 100),
                                           random_int(-100, 100),
                                           (BodyConfig){ .color = color, .line_width = 1.0f });
            food = arr_push(food, &single_food);
        }
    }
    return food;
}


void vehicle_eatFood(Vehicle* vehic, Body* food, bool is_badfood) {
    float mindist = INFINITY;
    int idx = -1;

    for (size_t i = 0; i < arr_len(food); i++) {
        float filter = is_badfood ? vehic->dna[2] : vehic->dna[3];
        float distance = vec3_distance(vehic->body.pos, food[i].pos);
        if ( distance < filter && distance < mindist) {
            mindist = distance;
            idx = i;
        }
    }

    if (idx > -1) {
        if (vec3_distance(vehic->body.pos, food[idx].pos) < 3) {
            arr_pop(food, idx);  //eat food
            if (is_badfood) {
                 vehic->health -= 0.1;
             } else {
                 vehic->health += 0.1;
             }
        } else {
            vehic_seek(vehic, food[idx].pos, is_badfood);
        }
    }
}

void vehic_boundary(Vehicle* vehic) {
    // Definiere deine Weltgrenzen
    const float minX = -130.0f;
    const float maxX = 130.0f;
    const float minY = -130.0f;
    const float maxY = 130.0f;
    const float minZ = -130.0f;
    const float maxZ = 130.0f;

    // X-Achse
    if (vehic->body.pos.x < minX || vehic->body.pos.x > maxX) {
        vehic->vel.x *= -1.0f;
    }

    // Y-Achse (Höhe)
    if (vehic->body.pos.y < minY || vehic->body.pos.y > maxY) {
        vehic->vel.y *= -1.0f;
    }

    // Z-Achse (Tiefe)
    if (vehic->body.pos.z < minZ || vehic->body.pos.z > maxZ) {
        vehic->vel.z *= -1.0f;
    }
}

bool vehic_isdead(Vehicle* vehic) {
    if (vehic->health < 0.0) {
        return true;
    } else {
        return false;
    }
}

int main(void) {
    if (!render_init(1600, 1000)) return 1;
    Vec3 cam_pos = vec3(50, 100, 200);
    Vec3 target  = vec3(0, 0, 0);
    Vec3 up      = vec3(0, 1, 0);
    render_set_fog(100.0f, 400.0f, 0.25f, 0.25f, 0.25f, 1.0f);

    random_init();

    Solid *grid_mesh = solid_grid(600, 24);
    Body grid = body_create(grid_mesh, 0, 0, 0, (BodyConfig){ .color = "#777774", .line_width = 1.0f });

    Solid *food_mesh = solid_sphere(3, 8, 8);
    Body *poison = food_create(30,"#FF0000", food_mesh);
    Body *food = food_create(30, "#44ff44", food_mesh);

    Solid *vehic_mesh = solid_pyramid(2, 6);
    Vehicle *vehics = arr_create(0, sizeof(Vehicle));

    for (size_t i = 0; i < 10; i++) {
        Vehicle vehic = vehic_create(body_create(vehic_mesh, 0, 20, 100, BODY_CONFIG_DEFAULT));
        vehic.vel = vec3(random_float(-2, 2), random_float(-2, 2), random_float(-2, 2));
        vehics = arr_push(vehics, &vehic);
    }

    while (!render_should_close()) {
        render_frame_begin();
        render_background(40, 40, 40);

        Mat4x4 view = mat4x4_lookat(cam_pos, target, up);
        Mat4x4 proj = mat4x4_perspective(1.2f, render_get_aspect(), 0.1f, 1000.0f);
        render_set_projection(&proj);

        body_draw(&grid, &view);

        food = food_respawn(food, food_mesh, 20, 30, "#44ff44");
        poison = food_respawn(poison, food_mesh, 20, 30, "#FF0000");

        for (size_t i = 0; i < arr_len(vehics); i++) {
            vehic_boundary(&vehics[i]);
            vehicle_eatFood(&vehics[i], food, false);
            vehicle_eatFood(&vehics[i], poison, true);
            //printf("Gesundheit: [%li] %f \n", i, vehics[i].health);
            vehic_alignToVelocity(&vehics[i]);
            vehic_update(&vehics[i]);
            
            if (vehics[i].health < 0.5f) {
                strcpy(vehics[i].body.color, "#FF0000");
            } else {
                strcpy(vehics[i].body.color, "#ffffff");
            }

            body_draw(&vehics[i].body, &view);
        }
        
        bool get_older = false;
        if (random_float(0, 1) < 0.015) {
            get_older = true;
        }

        for (int i = arr_len(vehics)-1; i >= 0; i--) {

            if (get_older) {
                vehics[i].health -= 0.05;
            }

            if (vehic_isdead(&vehics[i])) {
                arr_pop(vehics,i);
            }
        }

        for (size_t i = 0; i < arr_len(poison); i++) {
            body_draw(&poison[i], &view);
        }

        for (size_t i = 0; i < arr_len(food); i++) {
            body_draw(&food[i], &view);
        }

        render_frame_end();
    }

    //Programm-Ende
    // ALles zurücksetzen
    body_destroy(&grid);
    for (size_t i = 0; i < arr_len(poison); i++)
        body_destroy(&poison[i]);
    for (size_t i = 0; i < arr_len(food); i++)
        body_destroy(&food[i]);
    for (size_t i = 0; i < arr_len(vehics); i++)
        vehic_destroy(&vehics[i]);

    arr_free(poison);
    arr_free(food);
    arr_free(vehics);

    glfwTerminate();
    return 0;
}
