#include "navi_config.h"
#include "recovery_manager.h"

namespace mapnavigator
{

bool RecoveryManager::Step(
    MotionController* motion_controller,
    NavigationSession* session,
    NavigationRuntimeState* runtime_state,
    const PoseEstimate&,
    const RouteTrackingState& route,
    int64_t stalled_ms)
{
    if (motion_controller == nullptr || session == nullptr || runtime_state == nullptr) {
        return false;
    }

    if (stalled_ms < kObstacleRecoveryMinTriggerMs) {
        return false;
    }

    if (runtime_state->recovery.armed) {
        return false;
    }

    if (runtime_state->recovery.stuck_start_time.time_since_epoch().count() == 0) {
        runtime_state->recovery.stuck_start_time = std::chrono::steady_clock::now();
        runtime_state->recovery.stuck_anchor_distance = route.progress_distance;
    }

    motion_controller->SetForwardState(false);
    motion_controller->SetAction(LocalDriverAction::JumpForward, true);

    runtime_state->recovery.armed = true;
    session->ResetProgress();
    return true;
}

} // namespace mapnavigator
