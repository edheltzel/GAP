use bytes::Bytes;

use crate::api::schema::{
    EventData, EventEnvelope, EventKind, PaneClearAgentAuthorityParams, PaneDirection,
    PaneEdgesParams, PaneEdgesResult, PaneFocusDirectionParams, PaneFocusDirectionReason,
    PaneFocusDirectionResult, PaneLayoutPane, PaneLayoutParams, PaneLayoutRect, PaneLayoutSnapshot,
    PaneLayoutSplit, PaneListParams, PaneNeighborParams, PaneNeighborResult, PaneReadParams,
    PaneReadResult, PaneReleaseAgentParams, PaneRenameParams, PaneReportAgentParams,
    PaneReportAgentSessionParams, PaneReportMetadataParams, PaneResizeParams, PaneResizeReason,
    PaneResizeResult, PaneSendInputParams, PaneSendKeysParams, PaneSendTextParams, PaneSplitParams,
    PaneSwapParams, PaneSwapReason, PaneSwapResult, PaneTarget, PaneZoomMode, PaneZoomParams,
    PaneZoomReason, PaneZoomResult, ReadFormat, ReadSource, ResponseResult,
};
use crate::app::actions::{PaneZoomCommand, PaneZoomNoopReason};
use crate::app::{App, Mode};
use crate::layout::{find_in_direction, NavDirection, PaneId};

use super::super::api_helpers::{
    detect_state_from_api, encode_api_keys, encode_api_text, normalize_custom_status,
    normalize_reported_agent_label,
};
use super::responses::{encode_error, encode_success};

impl App {
    pub(super) fn handle_pane_split(&mut self, id: String, params: PaneSplitParams) -> String {
        let target = if let Some(target_pane_id) = params.target_pane_id.as_deref() {
            self.parse_pane_id(target_pane_id)
        } else if let Some(workspace_id) = params.workspace_id.as_deref() {
            self.parse_workspace_id(workspace_id).and_then(|ws_idx| {
                let pane_id = self.state.workspaces.get(ws_idx)?.focused_pane_id()?;
                Some((ws_idx, pane_id))
            })
        } else {
            self.state.active.and_then(|ws_idx| {
                let pane_id = self.state.workspaces.get(ws_idx)?.focused_pane_id()?;
                Some((ws_idx, pane_id))
            })
        };
        let Some((ws_idx, target_pane_id)) = target else {
            return encode_error(id, "pane_not_found", "pane not found");
        };
        let (rows, cols) = self.state.estimate_pane_size();
        let split_cwd = params.cwd.map(std::path::PathBuf::from).or_else(|| {
            let follow_cwd = self.state.workspaces.get(ws_idx).and_then(|ws| {
                let tab_idx = ws.find_tab_index_for_pane(target_pane_id)?;
                ws.tabs.get(tab_idx)?.cwd_for_pane(
                    target_pane_id,
                    &self.state.terminals,
                    &self.terminal_runtimes,
                )
            });
            Some(self.resolve_new_terminal_cwd(follow_cwd))
        });
        let default_shell = self.state.default_shell.clone();
        let scrollback_limit_bytes = self.state.pane_scrollback_limit_bytes;
        let host_terminal_theme = self.state.host_terminal_theme;
        let previous_focus = self.state.current_pane_focus_target();
        let Some(ws) = self.state.workspaces.get_mut(ws_idx) else {
            return encode_error(id, "pane_not_found", "pane not found");
        };
        let direction = match params.direction {
            crate::api::schema::SplitDirection::Right => ratatui::layout::Direction::Horizontal,
            crate::api::schema::SplitDirection::Down => ratatui::layout::Direction::Vertical,
        };
        let shell_config = crate::pane::PaneShellConfig::new(&default_shell, self.state.shell_mode);
        let split_result = match params.ratio {
            Some(ratio) => ws.split_pane_with_ratio(
                target_pane_id,
                direction,
                ratio,
                rows,
                cols,
                split_cwd,
                scrollback_limit_bytes,
                host_terminal_theme,
                shell_config,
                params.focus,
            ),
            None => ws.split_pane(
                target_pane_id,
                direction,
                rows,
                cols,
                split_cwd,
                scrollback_limit_bytes,
                host_terminal_theme,
                shell_config,
                params.focus,
            ),
        };
        let (target_tab_idx, new_pane) = match split_result {
            Some(Ok(result)) => result,
            Some(Err(err)) => return encode_error(id, "pane_split_failed", err.to_string()),
            None => return encode_error(id, "pane_not_found", "pane not found"),
        };
        if params.focus {
            self.state.switch_workspace_tab(ws_idx, target_tab_idx);
            self.state
                .record_pane_focus_change(previous_focus, ws_idx, new_pane.pane_id);
            self.state.mode = Mode::Terminal;
        }
        self.terminal_runtimes
            .insert(new_pane.terminal.id.clone(), new_pane.runtime);
        self.state
            .remove_alias_shadowed_by_new_pane(new_pane.pane_id);
        self.state
            .terminals
            .insert(new_pane.terminal.id.clone(), new_pane.terminal);
        self.schedule_session_save();
        let pane = self.pane_info(ws_idx, new_pane.pane_id).unwrap();
        self.emit_event(EventEnvelope {
            event: EventKind::PaneCreated,
            data: EventData::PaneCreated { pane: pane.clone() },
        });

        encode_success(id, ResponseResult::PaneInfo { pane })
    }

    pub(super) fn handle_pane_list(&mut self, id: String, params: PaneListParams) -> String {
        match self.collect_panes_for_workspace(params.workspace_id.as_deref()) {
            Ok(panes) => encode_success(id, ResponseResult::PaneList { panes }),
            Err((code, message)) => encode_error(id, &code, message),
        }
    }

    pub(super) fn handle_pane_get(&mut self, id: String, target: PaneTarget) -> String {
        let Some((ws_idx, pane_id)) = self.parse_pane_id(&target.pane_id) else {
            return pane_not_found(id, &target.pane_id);
        };
        let Some(pane) = self.pane_info(ws_idx, pane_id) else {
            return pane_not_found(id, &target.pane_id);
        };

        encode_success(id, ResponseResult::PaneInfo { pane })
    }

    pub(super) fn handle_pane_layout(&mut self, id: String, params: PaneLayoutParams) -> String {
        let Some((ws_idx, pane_id)) = self.resolve_optional_pane(params.pane_id.as_deref()) else {
            return encode_error(id, "pane_not_found", "pane not found");
        };
        let Some(tab_idx) = self.state.workspaces[ws_idx].find_tab_index_for_pane(pane_id) else {
            return pane_not_found(
                id,
                &self.public_pane_id(ws_idx, pane_id).unwrap_or_default(),
            );
        };
        let Some(layout) = self.pane_layout_snapshot(ws_idx, tab_idx) else {
            return encode_error(id, "pane_layout_unavailable", "pane layout unavailable");
        };

        encode_success(id, ResponseResult::PaneLayout { layout })
    }

    pub(super) fn handle_pane_neighbor(
        &mut self,
        id: String,
        params: PaneNeighborParams,
    ) -> String {
        let Some((ws_idx, pane_id)) = self.resolve_optional_pane(params.pane_id.as_deref()) else {
            return encode_error(id, "pane_not_found", "pane not found");
        };
        let Some(tab_idx) = self.state.workspaces[ws_idx].find_tab_index_for_pane(pane_id) else {
            return pane_not_found(
                id,
                &self.public_pane_id(ws_idx, pane_id).unwrap_or_default(),
            );
        };
        let Some(source_public_id) = self.public_pane_id(ws_idx, pane_id) else {
            return encode_error(id, "pane_not_found", "pane not found");
        };
        let neighbor_pane_id = self
            .directional_pane_target(ws_idx, tab_idx, pane_id, params.direction)
            .and_then(|pane_id| self.public_pane_id(ws_idx, pane_id));
        let Some(layout) = self.pane_layout_snapshot(ws_idx, tab_idx) else {
            return encode_error(id, "pane_layout_unavailable", "pane layout unavailable");
        };

        encode_success(
            id,
            ResponseResult::PaneNeighbor {
                neighbor: PaneNeighborResult {
                    pane_id: source_public_id,
                    direction: params.direction,
                    neighbor_pane_id,
                    layout,
                },
            },
        )
    }

    pub(super) fn handle_pane_edges(&mut self, id: String, params: PaneEdgesParams) -> String {
        let Some((ws_idx, pane_id)) = self.resolve_optional_pane(params.pane_id.as_deref()) else {
            return encode_error(id, "pane_not_found", "pane not found");
        };
        let Some(tab_idx) = self.state.workspaces[ws_idx].find_tab_index_for_pane(pane_id) else {
            return pane_not_found(
                id,
                &self.public_pane_id(ws_idx, pane_id).unwrap_or_default(),
            );
        };
        let Some(tab) = self
            .state
            .workspaces
            .get(ws_idx)
            .and_then(|ws| ws.tabs.get(tab_idx))
        else {
            return encode_error(id, "pane_layout_unavailable", "pane layout unavailable");
        };
        let area = self.state.view.terminal_area;
        let Some(info) = tab
            .layout
            .panes(area)
            .into_iter()
            .find(|info| info.id == pane_id)
        else {
            return pane_not_found(
                id,
                &self.public_pane_id(ws_idx, pane_id).unwrap_or_default(),
            );
        };
        let Some(pane_public_id) = self.public_pane_id(ws_idx, pane_id) else {
            return encode_error(id, "pane_not_found", "pane not found");
        };
        let Some(layout) = self.pane_layout_snapshot(ws_idx, tab_idx) else {
            return encode_error(id, "pane_layout_unavailable", "pane layout unavailable");
        };

        encode_success(
            id,
            ResponseResult::PaneEdges {
                edges: PaneEdgesResult {
                    pane_id: pane_public_id,
                    left: info.rect.x <= area.x,
                    right: info.rect.x + info.rect.width >= area.x + area.width,
                    up: info.rect.y <= area.y,
                    down: info.rect.y + info.rect.height >= area.y + area.height,
                    layout,
                },
            },
        )
    }

    pub(super) fn handle_pane_focus_direction(
        &mut self,
        id: String,
        params: PaneFocusDirectionParams,
    ) -> String {
        let Some((ws_idx, source_pane_id)) = self.resolve_optional_pane(params.pane_id.as_deref())
        else {
            return encode_error(id, "pane_not_found", "pane not found");
        };
        let Some(tab_idx) = self.state.workspaces[ws_idx].find_tab_index_for_pane(source_pane_id)
        else {
            return pane_not_found(
                id,
                &self
                    .public_pane_id(ws_idx, source_pane_id)
                    .unwrap_or_default(),
            );
        };
        let Some(source_public_id) = self.public_pane_id(ws_idx, source_pane_id) else {
            return encode_error(id, "pane_not_found", "pane not found");
        };
        let target =
            self.directional_pane_target(ws_idx, tab_idx, source_pane_id, params.direction);
        let reason = target
            .is_none()
            .then_some(PaneFocusDirectionReason::NoNeighbor);

        if let Some(target_pane_id) = target {
            self.state.focus_pane_in_workspace(ws_idx, target_pane_id);
            self.state.switch_workspace_tab(ws_idx, tab_idx);
            self.state.mode = Mode::Terminal;
        }
        let focused_pane_id = self
            .state
            .workspaces
            .get(ws_idx)
            .and_then(|ws| ws.tabs.get(tab_idx))
            .map(|tab| tab.layout.focused())
            .and_then(|pane_id| self.public_pane_id(ws_idx, pane_id));
        let Some(layout) = self.pane_layout_snapshot(ws_idx, tab_idx) else {
            return encode_error(id, "pane_layout_unavailable", "pane layout unavailable");
        };

        encode_success(
            id,
            ResponseResult::PaneFocusDirection {
                focus: PaneFocusDirectionResult {
                    changed: target.is_some(),
                    reason,
                    source_pane_id: source_public_id,
                    focused_pane_id,
                    layout,
                },
            },
        )
    }

    pub(super) fn handle_pane_resize(&mut self, id: String, params: PaneResizeParams) -> String {
        let Some((ws_idx, pane_id)) = self.resolve_optional_pane(params.pane_id.as_deref()) else {
            return encode_error(id, "pane_not_found", "pane not found");
        };
        let Some(tab_idx) = self.state.workspaces[ws_idx].find_tab_index_for_pane(pane_id) else {
            return pane_not_found(
                id,
                &self.public_pane_id(ws_idx, pane_id).unwrap_or_default(),
            );
        };
        let Some(pane_public_id) = self.public_pane_id(ws_idx, pane_id) else {
            return encode_error(id, "pane_not_found", "pane not found");
        };

        let amount = params
            .amount
            .filter(|amount| amount.is_finite())
            .unwrap_or(0.05)
            .abs()
            .min(0.5);
        let direction: NavDirection = params.direction.into();
        let area = self.state.view.terminal_area;
        let changed = self
            .state
            .workspaces
            .get_mut(ws_idx)
            .and_then(|ws| ws.tabs.get_mut(tab_idx))
            .is_some_and(|tab| tab.layout.resize_pane(pane_id, direction, amount, area));
        if changed {
            self.schedule_session_save();
        }

        let Some(layout) = self.pane_layout_snapshot(ws_idx, tab_idx) else {
            return encode_error(id, "pane_layout_unavailable", "pane layout unavailable");
        };
        let focused_pane_id = layout.focused_pane_id.clone();

        encode_success(
            id,
            ResponseResult::PaneResize {
                resize: PaneResizeResult {
                    changed,
                    reason: (!changed).then_some(PaneResizeReason::Unchanged),
                    pane_id: pane_public_id,
                    focused_pane_id,
                    layout,
                },
            },
        )
    }

    pub(super) fn handle_pane_swap(&mut self, id: String, params: PaneSwapParams) -> String {
        let directional = params.direction.is_some();
        let explicit = params.source_pane_id.is_some() || params.target_pane_id.is_some();
        if directional == explicit {
            return encode_error(
                id,
                "invalid_pane_swap",
                "provide either direction with optional pane_id, or source_pane_id and target_pane_id",
            );
        }

        let (ws_idx, tab_idx, source_pane_id, target_pane_id, reason) = if let Some(direction) =
            params.direction
        {
            let Some((ws_idx, source_pane_id)) =
                self.resolve_swap_source(params.pane_id.as_deref())
            else {
                return encode_error(id, "pane_not_found", "source pane not found");
            };
            let Some(tab_idx) =
                self.state.workspaces[ws_idx].find_tab_index_for_pane(source_pane_id)
            else {
                return pane_not_found(
                    id,
                    &self
                        .public_pane_id(ws_idx, source_pane_id)
                        .unwrap_or_default(),
                );
            };
            let target = self.directional_pane_target(ws_idx, tab_idx, source_pane_id, direction);
            match target {
                Some(target_pane_id) => {
                    (ws_idx, tab_idx, source_pane_id, Some(target_pane_id), None)
                }
                None => (
                    ws_idx,
                    tab_idx,
                    source_pane_id,
                    None,
                    Some(PaneSwapReason::NoNeighbor),
                ),
            }
        } else {
            let Some(source_raw) = params.source_pane_id.as_deref() else {
                return encode_error(id, "invalid_pane_swap", "missing source_pane_id");
            };
            let Some(target_raw) = params.target_pane_id.as_deref() else {
                return encode_error(id, "invalid_pane_swap", "missing target_pane_id");
            };
            let source = self
                .parse_pane_id(source_raw)
                .and_then(|(ws_idx, pane_id)| {
                    let tab_idx = self.state.workspaces[ws_idx].find_tab_index_for_pane(pane_id)?;
                    Some((ws_idx, tab_idx, pane_id))
                });
            let target = self
                .parse_pane_id(target_raw)
                .and_then(|(ws_idx, pane_id)| {
                    let tab_idx = self.state.workspaces[ws_idx].find_tab_index_for_pane(pane_id)?;
                    Some((ws_idx, tab_idx, pane_id))
                });
            let response_context = source
                .map(|(ws_idx, tab_idx, _)| (ws_idx, tab_idx))
                .or_else(|| target.map(|(ws_idx, tab_idx, _)| (ws_idx, tab_idx)))
                .or_else(|| {
                    let ws_idx = self.state.active?;
                    let tab_idx = self.state.workspaces.get(ws_idx)?.active_tab_index();
                    Some((ws_idx, tab_idx))
                });
            let Some((ws_idx, tab_idx)) = response_context else {
                return encode_error(id, "pane_layout_unavailable", "pane layout unavailable");
            };
            let source_pane_id = source
                .map(|(_, _, pane_id)| pane_id)
                .or_else(|| {
                    self.state
                        .workspaces
                        .get(ws_idx)?
                        .tabs
                        .get(tab_idx)
                        .map(|tab| tab.layout.focused())
                })
                .unwrap_or(PaneId::from_raw(0));
            let target_pane_id = target.map(|(_, _, pane_id)| pane_id);
            let reason = match (source, target) {
                (None, _) | (_, None) => Some(PaneSwapReason::NotFound),
                (Some((_, _, source)), Some((_, _, target))) if source == target => {
                    Some(PaneSwapReason::SamePane)
                }
                (Some((source_ws, source_tab, _)), Some((target_ws, target_tab, _)))
                    if source_ws != target_ws || source_tab != target_tab =>
                {
                    Some(PaneSwapReason::CrossTab)
                }
                _ => None,
            };
            (ws_idx, tab_idx, source_pane_id, target_pane_id, reason)
        };

        let mut changed = false;
        if reason.is_none() {
            if let Some(target_pane_id) = target_pane_id {
                let previous_focus = self.state.current_pane_focus_target();
                if let Some(tab) = self
                    .state
                    .workspaces
                    .get_mut(ws_idx)
                    .and_then(|ws| ws.tabs.get_mut(tab_idx))
                {
                    changed = tab.layout.swap_panes(source_pane_id, target_pane_id);
                    tab.layout.focus_pane(source_pane_id);
                    if changed {
                        self.state.switch_workspace_tab(ws_idx, tab_idx);
                        self.state
                            .record_pane_focus_change(previous_focus, ws_idx, source_pane_id);
                        self.state.mark_session_dirty();
                        self.schedule_session_save();
                    }
                }
            }
        }

        let source_public_id = match params.source_pane_id {
            Some(raw) => self
                .parse_pane_id(&raw)
                .and_then(|(idx, pane_id)| {
                    self.state
                        .workspaces
                        .get(idx)?
                        .find_tab_index_for_pane(pane_id)?;
                    self.public_pane_id(idx, pane_id)
                })
                .unwrap_or(raw),
            None => self
                .public_pane_id(ws_idx, source_pane_id)
                .unwrap_or_default(),
        };
        let target_public_id = match params.target_pane_id {
            Some(raw) => self
                .parse_pane_id(&raw)
                .and_then(|(idx, pane_id)| {
                    self.state
                        .workspaces
                        .get(idx)?
                        .find_tab_index_for_pane(pane_id)?;
                    self.public_pane_id(idx, pane_id)
                })
                .or(Some(raw)),
            None => target_pane_id.and_then(|pane_id| self.public_pane_id(ws_idx, pane_id)),
        };
        let Some(layout) = self.pane_layout_snapshot(ws_idx, tab_idx) else {
            return encode_error(id, "pane_layout_unavailable", "pane layout unavailable");
        };
        let focused_pane_id = layout.focused_pane_id.clone();

        encode_success(
            id,
            ResponseResult::PaneSwap {
                swap: PaneSwapResult {
                    changed,
                    reason,
                    source_pane_id: source_public_id,
                    target_pane_id: target_public_id,
                    focused_pane_id,
                    layout,
                },
            },
        )
    }

    pub(super) fn handle_pane_zoom(&mut self, id: String, params: PaneZoomParams) -> String {
        let Some((ws_idx, pane_id)) = self.resolve_optional_pane(params.pane_id.as_deref()) else {
            return encode_error(id, "pane_not_found", "pane not found");
        };
        let Some(tab_idx) = self.state.workspaces[ws_idx].find_tab_index_for_pane(pane_id) else {
            return pane_not_found(
                id,
                &self.public_pane_id(ws_idx, pane_id).unwrap_or_default(),
            );
        };
        let Some(pane_public_id) = self.public_pane_id(ws_idx, pane_id) else {
            return encode_error(id, "pane_not_found", "pane not found");
        };
        let command = match params.mode {
            PaneZoomMode::Toggle => PaneZoomCommand::Toggle,
            PaneZoomMode::On => PaneZoomCommand::On,
            PaneZoomMode::Off => PaneZoomCommand::Off,
        };
        let Some(outcome) = self.state.apply_pane_zoom(ws_idx, pane_id, command) else {
            return pane_not_found(id, &pane_public_id);
        };
        if outcome.changed || outcome.focus_changed {
            self.schedule_session_save();
        }
        self.state.mode = Mode::Terminal;
        let Some(layout) = self.pane_layout_snapshot(ws_idx, tab_idx) else {
            return encode_error(id, "pane_layout_unavailable", "pane layout unavailable");
        };
        let focused_pane_id = layout.focused_pane_id.clone();

        encode_success(
            id,
            ResponseResult::PaneZoom {
                zoom: PaneZoomResult {
                    changed: outcome.changed || outcome.focus_changed,
                    zoom_changed: outcome.changed,
                    focus_changed: outcome.focus_changed,
                    reason: outcome.reason.map(|reason| match reason {
                        PaneZoomNoopReason::SinglePane => PaneZoomReason::SinglePane,
                        PaneZoomNoopReason::AlreadyZoomed => PaneZoomReason::AlreadyZoomed,
                        PaneZoomNoopReason::AlreadyUnzoomed => PaneZoomReason::AlreadyUnzoomed,
                    }),
                    pane_id: pane_public_id,
                    focused_pane_id,
                    zoomed: outcome.zoomed,
                    layout,
                },
            },
        )
    }

    pub(super) fn handle_pane_rename(&mut self, id: String, params: PaneRenameParams) -> String {
        let Some((ws_idx, pane_id)) = self.parse_pane_id(&params.pane_id) else {
            return pane_not_found(id, &params.pane_id);
        };
        let Some(terminal_id) = self
            .state
            .workspaces
            .get(ws_idx)
            .and_then(|ws| ws.terminal_id(pane_id))
            .cloned()
        else {
            return pane_not_found(id, &params.pane_id);
        };
        let Some(terminal) = self.state.terminals.get_mut(&terminal_id) else {
            return pane_not_found(id, &params.pane_id);
        };
        match params.label.map(|label| label.trim().to_string()) {
            Some(label) if !label.is_empty() => terminal.set_manual_label(label),
            _ => terminal.clear_manual_label(),
        }
        self.state.mark_session_dirty();
        let pane = self.pane_info(ws_idx, pane_id).unwrap();

        encode_success(id, ResponseResult::PaneInfo { pane })
    }

    pub(super) fn handle_pane_read(&mut self, id: String, params: PaneReadParams) -> String {
        let Some((ws_idx, pane_id)) = self.parse_pane_id(&params.pane_id) else {
            return pane_not_found(id, &params.pane_id);
        };
        let Some((pane, workspace_id)) = self.lookup_runtime(ws_idx, pane_id) else {
            return pane_not_found(id, &params.pane_id);
        };
        let Some(tab_idx) = self
            .state
            .workspaces
            .get(ws_idx)
            .and_then(|ws| ws.find_tab_index_for_pane(pane_id))
        else {
            return pane_not_found(id, &params.pane_id);
        };
        let requested_lines = params.lines.unwrap_or(80).min(1000) as usize;
        let text = match params.format {
            ReadFormat::Text => match params.source {
                ReadSource::Visible => pane.visible_text(),
                ReadSource::Recent => pane.recent_text(requested_lines),
                ReadSource::RecentUnwrapped => pane.recent_unwrapped_text(requested_lines),
                ReadSource::Detection => pane.detection_text(),
            },
            ReadFormat::Ansi => match params.source {
                ReadSource::Visible => pane.visible_ansi(),
                ReadSource::Recent => pane.recent_ansi(requested_lines),
                ReadSource::RecentUnwrapped => pane.recent_unwrapped_ansi(requested_lines),
                ReadSource::Detection => pane.detection_text(),
            },
        };

        encode_success(
            id,
            ResponseResult::PaneRead {
                read: PaneReadResult {
                    pane_id: params.pane_id,
                    workspace_id,
                    tab_id: self.public_tab_id(ws_idx, tab_idx).unwrap(),
                    source: params.source,
                    format: params.format,
                    text,
                    revision: 0,
                    truncated: false,
                },
            },
        )
    }

    pub(super) fn handle_pane_report_agent(
        &mut self,
        id: String,
        params: PaneReportAgentParams,
    ) -> String {
        let Some((_ws_idx, pane_id)) = self.parse_pane_id(&params.pane_id) else {
            return pane_not_found(id, &params.pane_id);
        };
        let Some(agent_label) = normalize_reported_agent_label(&params.agent) else {
            return invalid_agent(id);
        };
        self.handle_internal_event(crate::events::AppEvent::HookStateReported {
            pane_id,
            session_ref: crate::agent_resume::session_ref_from_report(
                &params.source,
                &agent_label,
                params.agent_session_id,
                params.agent_session_path,
            ),
            source: params.source,
            agent_label,
            state: detect_state_from_api(params.state),
            message: params.message,
            custom_status: normalize_custom_status(params.custom_status),
            seq: params.seq,
        });

        encode_success(id, ResponseResult::Ok {})
    }

    pub(super) fn handle_pane_report_agent_session(
        &mut self,
        id: String,
        params: PaneReportAgentSessionParams,
    ) -> String {
        let Some((_ws_idx, pane_id)) = self.parse_pane_id(&params.pane_id) else {
            return pane_not_found(id, &params.pane_id);
        };
        let Some(agent_label) = normalize_reported_agent_label(&params.agent) else {
            return invalid_agent(id);
        };
        self.handle_internal_event(crate::events::AppEvent::AgentSessionReported {
            pane_id,
            session_ref: crate::agent_resume::session_ref_from_report(
                &params.source,
                &agent_label,
                params.agent_session_id,
                params.agent_session_path,
            ),
            source: params.source,
            agent_label,
            seq: params.seq,
        });

        encode_success(id, ResponseResult::Ok {})
    }

    pub(super) fn handle_pane_report_metadata(
        &mut self,
        id: String,
        params: PaneReportMetadataParams,
    ) -> String {
        let Some((_ws_idx, pane_id)) = self.parse_pane_id(&params.pane_id) else {
            return pane_not_found(id, &params.pane_id);
        };
        let agent_label = match params.agent.as_deref() {
            Some(agent) => match normalize_reported_agent_label(agent) {
                Some(agent_label) => Some(agent_label),
                None => return invalid_agent(id),
            },
            None => None,
        };
        let Some(source) = normalize_optional_text(Some(params.source)) else {
            return encode_error(id, "invalid_metadata_request", "missing metadata source");
        };
        let raw_title_set = params.title.is_some();
        let raw_display_agent_set = params.display_agent.is_some();
        let raw_custom_status_set = params.custom_status.is_some();
        let raw_state_labels_set = !params.state_labels.is_empty();
        let ttl = params.ttl_ms.map(std::time::Duration::from_millis);
        let title = normalize_presentation_text(params.title);
        let display_agent = normalize_presentation_text(params.display_agent);
        let custom_status = normalize_custom_status(params.custom_status);
        let applies_to_source = match params.applies_to_source {
            Some(applies_to_source) => {
                let Some(applies_to_source) = normalize_optional_text(Some(applies_to_source))
                else {
                    return encode_error(
                        id,
                        "invalid_metadata_request",
                        "missing metadata authority source",
                    );
                };
                Some(applies_to_source)
            }
            None => None,
        };
        let state_labels = match normalize_state_labels(params.state_labels) {
            Ok(labels) => labels,
            Err(status) => {
                return encode_error(
                    id,
                    "invalid_state_label",
                    format!("unknown state label: {status}"),
                );
            }
        };
        if raw_title_set && params.clear_title
            || raw_display_agent_set && params.clear_display_agent
            || raw_custom_status_set && params.clear_custom_status
            || raw_state_labels_set && params.clear_state_labels
        {
            return encode_error(
                id,
                "invalid_metadata_request",
                "cannot set and clear the same metadata field",
            );
        }
        if title.is_none()
            && display_agent.is_none()
            && custom_status.is_none()
            && state_labels.is_empty()
            && !params.clear_title
            && !params.clear_display_agent
            && !params.clear_custom_status
            && !params.clear_state_labels
        {
            return encode_error(
                id,
                "invalid_metadata_request",
                "missing metadata field to set or clear",
            );
        }
        self.handle_internal_event(crate::events::AppEvent::HookMetadataReported {
            pane_id,
            source,
            agent_label,
            applies_to_source,
            title,
            display_agent,
            custom_status,
            state_labels,
            clear_title: params.clear_title,
            clear_display_agent: params.clear_display_agent,
            clear_custom_status: params.clear_custom_status,
            clear_state_labels: params.clear_state_labels,
            seq: params.seq,
            ttl,
        });

        encode_success(id, ResponseResult::Ok {})
    }

    pub(super) fn handle_pane_clear_agent_authority(
        &mut self,
        id: String,
        params: PaneClearAgentAuthorityParams,
    ) -> String {
        let Some((_ws_idx, pane_id)) = self.parse_pane_id(&params.pane_id) else {
            return pane_not_found(id, &params.pane_id);
        };
        self.handle_internal_event(crate::events::AppEvent::HookAuthorityCleared {
            pane_id,
            source: params.source,
            seq: params.seq,
        });

        encode_success(id, ResponseResult::Ok {})
    }

    pub(super) fn handle_pane_release_agent(
        &mut self,
        id: String,
        params: PaneReleaseAgentParams,
    ) -> String {
        let Some((_ws_idx, pane_id)) = self.parse_pane_id(&params.pane_id) else {
            return pane_not_found(id, &params.pane_id);
        };
        let Some(agent_label) = normalize_reported_agent_label(&params.agent) else {
            return invalid_agent(id);
        };
        self.handle_internal_event(crate::events::AppEvent::HookAgentReleased {
            pane_id,
            source: params.source,
            known_agent: crate::detect::parse_agent_label(&agent_label),
            agent_label,
            seq: params.seq,
        });

        encode_success(id, ResponseResult::Ok {})
    }

    pub(super) fn handle_pane_send_text(
        &mut self,
        id: String,
        params: PaneSendTextParams,
    ) -> String {
        let Some((ws_idx, pane_id)) = self.parse_pane_id(&params.pane_id) else {
            return pane_not_found(id, &params.pane_id);
        };
        let Some(runtime) = self.lookup_runtime_sender(ws_idx, pane_id) else {
            return pane_not_found(id, &params.pane_id);
        };
        if let Err(err) = runtime.try_send_bytes(Bytes::from(params.text)) {
            return encode_error(id, "pane_send_failed", err.to_string());
        }

        encode_success(id, ResponseResult::Ok {})
    }

    pub(super) fn handle_pane_send_input(
        &mut self,
        id: String,
        params: PaneSendInputParams,
    ) -> String {
        let Some((ws_idx, pane_id)) = self.parse_pane_id(&params.pane_id) else {
            return pane_not_found(id, &params.pane_id);
        };
        let Some(runtime) = self.lookup_runtime_sender(ws_idx, pane_id) else {
            return pane_not_found(id, &params.pane_id);
        };
        let encoded_keys = match encode_api_keys(runtime, &params.keys) {
            Ok(encoded_keys) => encoded_keys,
            Err(key) => return encode_error(id, "invalid_key", format!("unsupported key {key}")),
        };
        if !params.text.is_empty() {
            let text_bytes = encode_api_text(runtime, &params.text);
            if let Err(err) = runtime.try_send_bytes(Bytes::from(text_bytes)) {
                return encode_error(id, "pane_send_failed", err.to_string());
            }
        }
        for bytes in encoded_keys {
            if let Err(err) = runtime.try_send_bytes(Bytes::from(bytes)) {
                return encode_error(id, "pane_send_failed", err.to_string());
            }
        }

        encode_success(id, ResponseResult::Ok {})
    }

    pub(super) fn handle_pane_close(&mut self, id: String, target: PaneTarget) -> String {
        let Some((ws_idx, pane_id)) = self.parse_pane_id(&target.pane_id) else {
            return pane_not_found(id, &target.pane_id);
        };
        if self.state.close_pane_would_close_workspace(ws_idx, pane_id)
            && self.state.confirm_implicit_worktree_group_close(ws_idx)
        {
            return encode_error(
                id,
                "confirmation_required",
                "closing this pane would close a worktree group",
            );
        }
        let workspace_id = self.state.workspaces[ws_idx].id.clone();
        let terminal_id = self.state.terminal_id_for_pane(ws_idx, pane_id);
        let should_close_workspace = {
            let Some(ws) = self.state.workspaces.get_mut(ws_idx) else {
                return pane_not_found(id, &target.pane_id);
            };
            ws.close_pane(pane_id)
        };
        if should_close_workspace {
            self.state.selected = ws_idx;
            self.state.close_selected_workspace();
            self.shutdown_detached_terminal_runtimes();
            self.emit_event(EventEnvelope {
                event: EventKind::PaneClosed,
                data: EventData::PaneClosed {
                    pane_id: target.pane_id.clone(),
                    workspace_id: workspace_id.clone(),
                },
            });
            self.emit_event(EventEnvelope {
                event: EventKind::WorkspaceClosed,
                data: EventData::WorkspaceClosed { workspace_id },
            });
        } else {
            self.state.remove_unattached_terminal_ids(terminal_id);
            self.shutdown_detached_terminal_runtimes();
            self.schedule_session_save();
            self.emit_event(EventEnvelope {
                event: EventKind::PaneClosed,
                data: EventData::PaneClosed {
                    pane_id: target.pane_id,
                    workspace_id,
                },
            });
        }

        encode_success(id, ResponseResult::Ok {})
    }

    pub(super) fn handle_pane_send_keys(
        &mut self,
        id: String,
        params: PaneSendKeysParams,
    ) -> String {
        let Some((ws_idx, pane_id)) = self.parse_pane_id(&params.pane_id) else {
            return pane_not_found(id, &params.pane_id);
        };
        let Some(runtime) = self.lookup_runtime_sender(ws_idx, pane_id) else {
            return pane_not_found(id, &params.pane_id);
        };
        let encoded_keys = match encode_api_keys(runtime, &params.keys) {
            Ok(encoded_keys) => encoded_keys,
            Err(key) => return encode_error(id, "invalid_key", format!("unsupported key {key}")),
        };
        for bytes in encoded_keys {
            if let Err(err) = runtime.try_send_bytes(Bytes::from(bytes)) {
                return encode_error(id, "pane_send_failed", err.to_string());
            }
        }

        encode_success(id, ResponseResult::Ok {})
    }
}

fn normalize_optional_text(value: Option<String>) -> Option<String> {
    let value = value?.trim().to_string();
    (!value.is_empty()).then_some(value)
}

fn normalize_presentation_text(value: Option<String>) -> Option<String> {
    let trimmed = value?.trim().to_string();
    let normalized: String = trimmed
        .chars()
        .filter(|ch| !ch.is_control())
        .take(80)
        .collect();
    (!normalized.trim().is_empty()).then(|| normalized.trim().to_string())
}

fn normalize_state_labels(
    labels: std::collections::HashMap<String, String>,
) -> Result<std::collections::HashMap<String, String>, String> {
    labels
        .into_iter()
        .map(|(status, label)| {
            let status = status.trim().to_ascii_lowercase();
            if !matches!(
                status.as_str(),
                "idle" | "working" | "blocked" | "done" | "unknown"
            ) {
                return Err(status);
            }
            Ok(normalize_presentation_text(Some(label)).map(|label| (status, label)))
        })
        .filter_map(Result::transpose)
        .collect()
}

fn pane_not_found(id: String, pane_id: &str) -> String {
    encode_error(id, "pane_not_found", format!("pane {pane_id} not found"))
}

impl App {
    fn resolve_optional_pane(&self, pane_id: Option<&str>) -> Option<(usize, PaneId)> {
        match pane_id {
            Some(pane_id) => self.parse_pane_id(pane_id),
            None => {
                let ws_idx = self.state.active?;
                let pane_id = self.state.workspaces.get(ws_idx)?.focused_pane_id()?;
                Some((ws_idx, pane_id))
            }
        }
    }

    fn resolve_swap_source(&self, pane_id: Option<&str>) -> Option<(usize, PaneId)> {
        self.resolve_optional_pane(pane_id)
    }

    fn directional_pane_target(
        &self,
        ws_idx: usize,
        tab_idx: usize,
        source_pane_id: PaneId,
        direction: PaneDirection,
    ) -> Option<PaneId> {
        let tab = self.state.workspaces.get(ws_idx)?.tabs.get(tab_idx)?;
        let panes = tab.layout.panes(self.state.view.terminal_area);
        let source = panes.iter().find(|pane| pane.id == source_pane_id)?;
        find_in_direction(source, direction.into(), &panes)
    }

    fn pane_layout_snapshot(&self, ws_idx: usize, tab_idx: usize) -> Option<PaneLayoutSnapshot> {
        let ws = self.state.workspaces.get(ws_idx)?;
        let tab = ws.tabs.get(tab_idx)?;
        let area = self.state.view.terminal_area;
        let focused_pane_id = self.public_pane_id(ws_idx, tab.layout.focused())?;
        let panes = tab
            .layout
            .panes(area)
            .into_iter()
            .filter_map(|pane| {
                Some(PaneLayoutPane {
                    pane_id: self.public_pane_id(ws_idx, pane.id)?,
                    focused: pane.is_focused,
                    rect: pane.rect.into(),
                })
            })
            .collect();
        let splits = tab
            .layout
            .splits(area)
            .into_iter()
            .enumerate()
            .map(|(idx, split)| PaneLayoutSplit {
                id: split_path_id(idx, &split.path),
                direction: match split.direction {
                    ratatui::layout::Direction::Horizontal => {
                        crate::api::schema::SplitDirection::Right
                    }
                    ratatui::layout::Direction::Vertical => {
                        crate::api::schema::SplitDirection::Down
                    }
                },
                ratio: split.ratio,
                rect: split.area.into(),
            })
            .collect();

        Some(PaneLayoutSnapshot {
            workspace_id: self.public_workspace_id(ws_idx),
            tab_id: self.public_tab_id(ws_idx, tab_idx)?,
            zoomed: tab.zoomed,
            area: area.into(),
            focused_pane_id,
            panes,
            splits,
        })
    }
}

impl From<PaneDirection> for NavDirection {
    fn from(direction: PaneDirection) -> Self {
        match direction {
            PaneDirection::Left => NavDirection::Left,
            PaneDirection::Right => NavDirection::Right,
            PaneDirection::Up => NavDirection::Up,
            PaneDirection::Down => NavDirection::Down,
        }
    }
}

impl From<ratatui::layout::Rect> for PaneLayoutRect {
    fn from(rect: ratatui::layout::Rect) -> Self {
        Self {
            x: rect.x,
            y: rect.y,
            width: rect.width,
            height: rect.height,
        }
    }
}

fn split_path_id(idx: usize, path: &[bool]) -> String {
    if path.is_empty() {
        return format!("split_{idx}_root");
    }
    let path = path
        .iter()
        .map(|right| if *right { "1" } else { "0" })
        .collect::<Vec<_>>()
        .join("");
    format!("split_{idx}_{path}")
}

fn invalid_agent(id: String) -> String {
    encode_error(id, "invalid_agent", "agent label must not be empty")
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{api::schema::SuccessResponse, config::Config, workspace::Workspace};

    fn app_with_linked_worktree() -> App {
        let (_api_tx, api_rx) = tokio::sync::mpsc::unbounded_channel();
        let mut app = App::new(
            &Config::default(),
            true,
            None,
            api_rx,
            crate::api::EventHub::default(),
        );
        app.state.workspaces = vec![Workspace::test_new("issue")];
        app.state.workspaces[0].worktree_space = Some(crate::workspace::WorktreeSpaceMembership {
            key: "repo-key".into(),
            label: "herdr".into(),
            repo_root: "/repo/herdr".into(),
            checkout_path: "/repo/herdr-issue".into(),
            is_linked_worktree: true,
        });
        app
    }

    #[test]
    fn api_pane_close_closes_linked_worktree_workspace_only() {
        let mut app = app_with_linked_worktree();
        let pane_id = app.state.workspaces[0].tabs[0].root_pane;
        let public_pane_id = app.public_pane_id(0, pane_id).unwrap();

        let response = app.handle_pane_close(
            "req".into(),
            PaneTarget {
                pane_id: public_pane_id,
            },
        );

        let success: SuccessResponse = serde_json::from_str(&response).unwrap();
        assert_eq!(success.id, "req");
        assert_eq!(app.state.request_remove_linked_worktree, None);
        assert!(app.state.workspaces.is_empty());
    }

    #[test]
    fn api_pane_swap_explicit_source_and_target_preserves_focus_and_returns_layout() {
        let mut app = app_with_linked_worktree();
        let source = app.state.workspaces[0].tabs[0].root_pane;
        let target = app.state.workspaces[0].test_split(ratatui::layout::Direction::Horizontal);
        app.state.workspaces[0].tabs[0].layout.focus_pane(source);
        crate::ui::compute_view(&mut app.state, ratatui::layout::Rect::new(0, 0, 100, 20));
        let source_public = app.public_pane_id(0, source).unwrap();
        let target_public = app.public_pane_id(0, target).unwrap();

        let response = app.handle_pane_swap(
            "req".into(),
            PaneSwapParams {
                source_pane_id: Some(source_public.clone()),
                target_pane_id: Some(target_public.clone()),
                ..PaneSwapParams::default()
            },
        );

        let success: SuccessResponse = serde_json::from_str(&response).unwrap();
        let ResponseResult::PaneSwap { swap } = success.result else {
            panic!("expected pane swap response");
        };
        assert!(swap.changed);
        assert_eq!(swap.reason, None);
        assert_eq!(swap.source_pane_id, source_public);
        assert_eq!(swap.target_pane_id, Some(target_public));
        assert_eq!(swap.focused_pane_id, swap.source_pane_id);
        assert_eq!(swap.layout.focused_pane_id, swap.source_pane_id);
        assert_eq!(swap.layout.panes.len(), 2);
        assert_eq!(app.state.workspaces[0].focused_pane_id(), Some(source));
    }

    #[test]
    fn api_pane_swap_unfocused_source_updates_last_pane_history() {
        let mut app = app_with_linked_worktree();
        let source = app.state.workspaces[0].tabs[0].root_pane;
        let focused = app.state.workspaces[0].test_split(ratatui::layout::Direction::Horizontal);
        let target = app.state.workspaces[0].test_split(ratatui::layout::Direction::Vertical);
        app.state.active = Some(0);
        app.state.selected = 0;
        app.state.workspaces[0].tabs[0].layout.focus_pane(focused);
        crate::ui::compute_view(&mut app.state, ratatui::layout::Rect::new(0, 0, 100, 20));
        let source_public = app.public_pane_id(0, source).unwrap();
        let target_public = app.public_pane_id(0, target).unwrap();

        let response = app.handle_pane_swap(
            "req".into(),
            PaneSwapParams {
                source_pane_id: Some(source_public),
                target_pane_id: Some(target_public),
                ..PaneSwapParams::default()
            },
        );

        let success: SuccessResponse = serde_json::from_str(&response).unwrap();
        let ResponseResult::PaneSwap { swap } = success.result else {
            panic!("expected pane swap response");
        };
        assert!(swap.changed);
        assert_eq!(app.state.workspaces[0].focused_pane_id(), Some(source));

        app.state.last_pane();

        assert_eq!(app.state.workspaces[0].focused_pane_id(), Some(focused));
    }

    #[test]
    fn api_pane_swap_direction_no_neighbor_returns_unchanged_layout() {
        let mut app = app_with_linked_worktree();
        let source = app.state.workspaces[0].tabs[0].root_pane;
        app.state.workspaces[0].tabs[0].layout.focus_pane(source);
        crate::ui::compute_view(&mut app.state, ratatui::layout::Rect::new(0, 0, 100, 20));
        let source_public = app.public_pane_id(0, source).unwrap();

        let response = app.handle_pane_swap(
            "req".into(),
            PaneSwapParams {
                pane_id: Some(source_public.clone()),
                direction: Some(PaneDirection::Left),
                ..PaneSwapParams::default()
            },
        );

        let success: SuccessResponse = serde_json::from_str(&response).unwrap();
        let ResponseResult::PaneSwap { swap } = success.result else {
            panic!("expected pane swap response");
        };
        assert!(!swap.changed);
        assert_eq!(swap.reason, Some(PaneSwapReason::NoNeighbor));
        assert_eq!(swap.source_pane_id, source_public);
        assert_eq!(swap.target_pane_id, None);
        assert_eq!(swap.layout.panes.len(), 1);
    }

    #[test]
    fn api_pane_swap_explicit_missing_target_returns_not_found_noop() {
        let mut app = app_with_linked_worktree();
        let source = app.state.workspaces[0].tabs[0].root_pane;
        let source_public = app.public_pane_id(0, source).unwrap();

        let response = app.handle_pane_swap(
            "req".into(),
            PaneSwapParams {
                source_pane_id: Some(source_public.clone()),
                target_pane_id: Some("missing-pane".into()),
                ..PaneSwapParams::default()
            },
        );

        let success: SuccessResponse = serde_json::from_str(&response).unwrap();
        let ResponseResult::PaneSwap { swap } = success.result else {
            panic!("expected pane swap response");
        };
        assert!(!swap.changed);
        assert_eq!(swap.reason, Some(PaneSwapReason::NotFound));
        assert_eq!(swap.source_pane_id, source_public);
        assert_eq!(swap.target_pane_id, Some("missing-pane".into()));
        assert_eq!(swap.layout.panes.len(), 1);
    }

    #[test]
    fn api_pane_swap_explicit_missing_source_returns_not_found_noop() {
        let mut app = app_with_linked_worktree();
        let target = app.state.workspaces[0].tabs[0].root_pane;
        let target_public = app.public_pane_id(0, target).unwrap();

        let response = app.handle_pane_swap(
            "req".into(),
            PaneSwapParams {
                source_pane_id: Some("missing-pane".into()),
                target_pane_id: Some(target_public.clone()),
                ..PaneSwapParams::default()
            },
        );

        let success: SuccessResponse = serde_json::from_str(&response).unwrap();
        let ResponseResult::PaneSwap { swap } = success.result else {
            panic!("expected pane swap response");
        };
        assert!(!swap.changed);
        assert_eq!(swap.reason, Some(PaneSwapReason::NotFound));
        assert_eq!(swap.source_pane_id, "missing-pane");
        assert_eq!(swap.target_pane_id, Some(target_public));
        assert_eq!(swap.layout.panes.len(), 1);
    }

    #[test]
    fn api_pane_swap_explicit_cross_workspace_preserves_target_id() {
        let mut app = app_with_linked_worktree();
        app.state.workspaces.push(Workspace::test_new("other"));
        let source = app.state.workspaces[0].tabs[0].root_pane;
        let target = app.state.workspaces[1].tabs[0].root_pane;
        let source_public = app.public_pane_id(0, source).unwrap();
        let target_public = app.public_pane_id(1, target).unwrap();

        let response = app.handle_pane_swap(
            "req".into(),
            PaneSwapParams {
                source_pane_id: Some(source_public.clone()),
                target_pane_id: Some(target_public.clone()),
                ..PaneSwapParams::default()
            },
        );

        let success: SuccessResponse = serde_json::from_str(&response).unwrap();
        let ResponseResult::PaneSwap { swap } = success.result else {
            panic!("expected pane swap response");
        };
        assert!(!swap.changed);
        assert_eq!(swap.reason, Some(PaneSwapReason::CrossTab));
        assert_eq!(swap.source_pane_id, source_public);
        assert_eq!(swap.target_pane_id, Some(target_public));
        assert_eq!(swap.layout.workspace_id, app.public_workspace_id(0));
    }

    #[test]
    fn api_pane_zoom_current_toggles_zoom() {
        let mut app = app_with_linked_worktree();
        app.state.active = Some(0);
        app.state.selected = 0;
        let root = app.state.workspaces[0].tabs[0].root_pane;
        let _right = app.state.workspaces[0].test_split(ratatui::layout::Direction::Horizontal);
        app.state.workspaces[0].tabs[0].layout.focus_pane(root);
        let root_public = app.public_pane_id(0, root).unwrap();

        let response = app.handle_pane_zoom("req".into(), PaneZoomParams::default());

        let success: SuccessResponse = serde_json::from_str(&response).unwrap();
        let ResponseResult::PaneZoom { zoom } = success.result else {
            panic!("expected pane zoom response");
        };
        assert!(zoom.changed);
        assert!(zoom.zoom_changed);
        assert!(!zoom.focus_changed);
        assert_eq!(zoom.reason, None);
        assert_eq!(zoom.pane_id, root_public);
        assert_eq!(zoom.focused_pane_id, zoom.pane_id);
        assert!(zoom.zoomed);
        assert!(zoom.layout.zoomed);

        let response = app.handle_pane_zoom("req".into(), PaneZoomParams::default());
        let success: SuccessResponse = serde_json::from_str(&response).unwrap();
        let ResponseResult::PaneZoom { zoom } = success.result else {
            panic!("expected pane zoom response");
        };
        assert!(zoom.changed);
        assert!(zoom.zoom_changed);
        assert!(!zoom.focus_changed);
        assert!(!zoom.zoomed);
        assert!(!zoom.layout.zoomed);
    }

    #[test]
    fn api_pane_zoom_explicit_background_pane_updates_focus_history() {
        let mut app = app_with_linked_worktree();
        app.state.workspaces.push(Workspace::test_new("other"));
        let first = app.state.workspaces[0].tabs[0].root_pane;
        let target = app.state.workspaces[1].tabs[0].root_pane;
        let _other = app.state.workspaces[1].test_split(ratatui::layout::Direction::Horizontal);
        app.state.active = Some(0);
        app.state.selected = 0;
        app.state.workspaces[0].tabs[0].layout.focus_pane(first);
        let target_public = app.public_pane_id(1, target).unwrap();

        let response = app.handle_pane_zoom(
            "req".into(),
            PaneZoomParams {
                pane_id: Some(target_public.clone()),
                mode: PaneZoomMode::On,
            },
        );

        let success: SuccessResponse = serde_json::from_str(&response).unwrap();
        let ResponseResult::PaneZoom { zoom } = success.result else {
            panic!("expected pane zoom response");
        };
        assert!(zoom.changed);
        assert!(zoom.zoom_changed);
        assert!(zoom.focus_changed);
        assert_eq!(zoom.pane_id, target_public);
        assert_eq!(app.state.active, Some(1));
        assert_eq!(app.state.workspaces[1].focused_pane_id(), Some(target));
        assert!(app.state.workspaces[1].tabs[0].zoomed);

        app.state.last_pane();

        assert_eq!(app.state.active, Some(0));
        assert_eq!(app.state.workspaces[0].focused_pane_id(), Some(first));
    }

    #[test]
    fn api_pane_zoom_single_pane_returns_noop() {
        let mut app = app_with_linked_worktree();
        app.state.active = Some(0);
        app.state.selected = 0;
        let root = app.state.workspaces[0].tabs[0].root_pane;
        let root_public = app.public_pane_id(0, root).unwrap();

        let response = app.handle_pane_zoom(
            "req".into(),
            PaneZoomParams {
                pane_id: Some(root_public.clone()),
                mode: PaneZoomMode::Toggle,
            },
        );

        let success: SuccessResponse = serde_json::from_str(&response).unwrap();
        let ResponseResult::PaneZoom { zoom } = success.result else {
            panic!("expected pane zoom response");
        };
        assert!(!zoom.changed);
        assert!(!zoom.zoom_changed);
        assert!(!zoom.focus_changed);
        assert_eq!(zoom.reason, Some(PaneZoomReason::SinglePane));
        assert_eq!(zoom.pane_id, root_public);
        assert!(!zoom.zoomed);
        assert!(!app.state.workspaces[0].tabs[0].zoomed);
    }

    #[test]
    fn api_pane_zoom_on_and_off_are_idempotent() {
        let mut app = app_with_linked_worktree();
        app.state.active = Some(0);
        app.state.selected = 0;
        let root = app.state.workspaces[0].tabs[0].root_pane;
        let _right = app.state.workspaces[0].test_split(ratatui::layout::Direction::Horizontal);
        app.state.workspaces[0].tabs[0].layout.focus_pane(root);
        let root_public = app.public_pane_id(0, root).unwrap();

        let response = app.handle_pane_zoom(
            "req".into(),
            PaneZoomParams {
                pane_id: Some(root_public.clone()),
                mode: PaneZoomMode::On,
            },
        );
        let success: SuccessResponse = serde_json::from_str(&response).unwrap();
        let ResponseResult::PaneZoom { zoom } = success.result else {
            panic!("expected pane zoom response");
        };
        assert!(zoom.changed);
        assert!(zoom.zoom_changed);
        assert!(!zoom.focus_changed);
        assert!(zoom.zoomed);

        let response = app.handle_pane_zoom(
            "req".into(),
            PaneZoomParams {
                pane_id: Some(root_public.clone()),
                mode: PaneZoomMode::On,
            },
        );
        let success: SuccessResponse = serde_json::from_str(&response).unwrap();
        let ResponseResult::PaneZoom { zoom } = success.result else {
            panic!("expected pane zoom response");
        };
        assert!(!zoom.changed);
        assert!(!zoom.zoom_changed);
        assert!(!zoom.focus_changed);
        assert_eq!(zoom.reason, Some(PaneZoomReason::AlreadyZoomed));
        assert!(zoom.zoomed);

        let response = app.handle_pane_zoom(
            "req".into(),
            PaneZoomParams {
                pane_id: Some(root_public),
                mode: PaneZoomMode::Off,
            },
        );
        let success: SuccessResponse = serde_json::from_str(&response).unwrap();
        let ResponseResult::PaneZoom { zoom } = success.result else {
            panic!("expected pane zoom response");
        };
        assert!(zoom.changed);
        assert!(zoom.zoom_changed);
        assert!(!zoom.focus_changed);
        assert!(!zoom.zoomed);

        let response = app.handle_pane_zoom(
            "req".into(),
            PaneZoomParams {
                pane_id: None,
                mode: PaneZoomMode::Off,
            },
        );
        let success: SuccessResponse = serde_json::from_str(&response).unwrap();
        let ResponseResult::PaneZoom { zoom } = success.result else {
            panic!("expected pane zoom response");
        };
        assert!(!zoom.changed);
        assert!(!zoom.zoom_changed);
        assert!(!zoom.focus_changed);
        assert_eq!(zoom.reason, Some(PaneZoomReason::AlreadyUnzoomed));
        assert!(!zoom.zoomed);
    }

    #[test]
    fn api_pane_zoom_idempotent_mode_reports_focus_change() {
        let mut app = app_with_linked_worktree();
        app.state.active = Some(0);
        app.state.selected = 0;
        let root = app.state.workspaces[0].tabs[0].root_pane;
        let right = app.state.workspaces[0].test_split(ratatui::layout::Direction::Horizontal);
        app.state.workspaces[0].tabs[0].layout.focus_pane(root);
        app.state.workspaces[0].tabs[0].zoomed = true;
        let right_public = app.public_pane_id(0, right).unwrap();

        let response = app.handle_pane_zoom(
            "req".into(),
            PaneZoomParams {
                pane_id: Some(right_public),
                mode: PaneZoomMode::On,
            },
        );

        let success: SuccessResponse = serde_json::from_str(&response).unwrap();
        let ResponseResult::PaneZoom { zoom } = success.result else {
            panic!("expected pane zoom response");
        };
        assert!(zoom.changed);
        assert!(!zoom.zoom_changed);
        assert!(zoom.focus_changed);
        assert_eq!(zoom.reason, Some(PaneZoomReason::AlreadyZoomed));
        assert!(zoom.zoomed);
        assert_eq!(app.state.workspaces[0].focused_pane_id(), Some(right));
    }

    #[test]
    fn api_pane_zoom_params_serialize_modes() {
        let request = crate::api::schema::Request {
            id: "req".into(),
            method: crate::api::schema::Method::PaneZoom(PaneZoomParams {
                pane_id: Some("issue-1".into()),
                mode: PaneZoomMode::On,
            }),
        };

        let encoded = serde_json::to_string(&request).unwrap();
        assert!(encoded.contains("\"method\":\"pane.zoom\""));
        assert!(encoded.contains("\"mode\":\"on\""));

        let decoded: crate::api::schema::Request = serde_json::from_str(&encoded).unwrap();
        let crate::api::schema::Method::PaneZoom(params) = decoded.method else {
            panic!("expected pane zoom request");
        };
        assert_eq!(params.pane_id, Some("issue-1".into()));
        assert_eq!(params.mode, PaneZoomMode::On);
    }

    #[test]
    fn api_pane_layout_returns_public_ids_rects_and_splits() {
        let mut app = app_with_linked_worktree();
        let root = app.state.workspaces[0].tabs[0].root_pane;
        let right = app.state.workspaces[0].test_split(ratatui::layout::Direction::Horizontal);
        app.state.workspaces[0].tabs[0].layout.focus_pane(root);
        crate::ui::compute_view(&mut app.state, ratatui::layout::Rect::new(0, 0, 100, 20));
        let root_public = app.public_pane_id(0, root).unwrap();
        let right_public = app.public_pane_id(0, right).unwrap();

        let response = app.handle_pane_layout(
            "req".into(),
            crate::api::schema::PaneLayoutParams {
                pane_id: Some(root_public.clone()),
            },
        );

        let success: SuccessResponse = serde_json::from_str(&response).unwrap();
        let ResponseResult::PaneLayout { layout } = success.result else {
            panic!("expected pane layout response");
        };
        assert_eq!(layout.focused_pane_id, root_public);
        assert!(layout.panes.iter().any(|pane| pane.pane_id == root_public));
        assert!(layout.panes.iter().any(|pane| pane.pane_id == right_public));
        assert_eq!(layout.splits.len(), 1);
        assert_eq!(
            layout.splits[0].direction,
            crate::api::schema::SplitDirection::Right
        );
    }

    #[test]
    fn api_pane_neighbor_returns_directional_neighbor_public_id() {
        let mut app = app_with_linked_worktree();
        let root = app.state.workspaces[0].tabs[0].root_pane;
        let right = app.state.workspaces[0].test_split(ratatui::layout::Direction::Horizontal);
        app.state.workspaces[0].tabs[0].layout.focus_pane(root);
        crate::ui::compute_view(&mut app.state, ratatui::layout::Rect::new(0, 0, 100, 20));
        let root_public = app.public_pane_id(0, root).unwrap();
        let right_public = app.public_pane_id(0, right).unwrap();

        let response = app.handle_pane_neighbor(
            "req".into(),
            crate::api::schema::PaneNeighborParams {
                pane_id: Some(root_public.clone()),
                direction: PaneDirection::Right,
            },
        );

        let success: SuccessResponse = serde_json::from_str(&response).unwrap();
        let ResponseResult::PaneNeighbor { neighbor } = success.result else {
            panic!("expected pane neighbor response");
        };
        assert_eq!(neighbor.pane_id, root_public);
        assert_eq!(neighbor.direction, PaneDirection::Right);
        assert_eq!(neighbor.neighbor_pane_id, Some(right_public));
    }

    #[test]
    fn api_pane_edges_reports_physical_layout_edges() {
        let mut app = app_with_linked_worktree();
        let root = app.state.workspaces[0].tabs[0].root_pane;
        let right = app.state.workspaces[0].test_split(ratatui::layout::Direction::Horizontal);
        app.state.workspaces[0].tabs[0].layout.focus_pane(root);
        crate::ui::compute_view(&mut app.state, ratatui::layout::Rect::new(0, 0, 100, 20));
        let right_public = app.public_pane_id(0, right).unwrap();

        let response = app.handle_pane_edges(
            "req".into(),
            crate::api::schema::PaneEdgesParams {
                pane_id: Some(right_public.clone()),
            },
        );

        let success: SuccessResponse = serde_json::from_str(&response).unwrap();
        let ResponseResult::PaneEdges { edges } = success.result else {
            panic!("expected pane edges response");
        };
        assert_eq!(edges.pane_id, right_public);
        assert!(!edges.left);
        assert!(edges.right);
        assert!(edges.up);
        assert!(edges.down);
    }

    #[test]
    fn api_pane_resize_changes_target_ratio_without_changing_focus() {
        let mut app = app_with_linked_worktree();
        let root = app.state.workspaces[0].tabs[0].root_pane;
        let right = app.state.workspaces[0].test_split(ratatui::layout::Direction::Horizontal);
        app.state.workspaces[0].tabs[0].layout.focus_pane(right);
        crate::ui::compute_view(&mut app.state, ratatui::layout::Rect::new(0, 0, 100, 20));
        let root_public = app.public_pane_id(0, root).unwrap();
        let right_public = app.public_pane_id(0, right).unwrap();

        let response = app.handle_pane_resize(
            "req".into(),
            crate::api::schema::PaneResizeParams {
                pane_id: Some(root_public.clone()),
                direction: PaneDirection::Right,
                amount: Some(0.1),
            },
        );

        let success: SuccessResponse = serde_json::from_str(&response).unwrap();
        let ResponseResult::PaneResize { resize } = success.result else {
            panic!("expected pane resize response");
        };
        assert!(resize.changed);
        assert_eq!(resize.reason, None);
        assert_eq!(resize.pane_id, root_public);
        assert_eq!(resize.focused_pane_id, right_public);
        assert_eq!(resize.layout.focused_pane_id, right_public);
        assert!((resize.layout.splits[0].ratio - 0.6).abs() < f32::EPSILON);
        assert_eq!(app.state.workspaces[0].focused_pane_id(), Some(right));
    }

    #[test]
    fn api_pane_focus_direction_focuses_neighbor() {
        let mut app = app_with_linked_worktree();
        let root = app.state.workspaces[0].tabs[0].root_pane;
        let right = app.state.workspaces[0].test_split(ratatui::layout::Direction::Horizontal);
        app.state.workspaces[0].tabs[0].layout.focus_pane(root);
        crate::ui::compute_view(&mut app.state, ratatui::layout::Rect::new(0, 0, 100, 20));
        let root_public = app.public_pane_id(0, root).unwrap();
        let right_public = app.public_pane_id(0, right).unwrap();

        let response = app.handle_pane_focus_direction(
            "req".into(),
            crate::api::schema::PaneFocusDirectionParams {
                pane_id: Some(root_public.clone()),
                direction: PaneDirection::Right,
            },
        );

        let success: SuccessResponse = serde_json::from_str(&response).unwrap();
        let ResponseResult::PaneFocusDirection { focus } = success.result else {
            panic!("expected pane focus direction response");
        };
        assert!(focus.changed);
        assert_eq!(focus.reason, None);
        assert_eq!(focus.source_pane_id, root_public);
        assert_eq!(focus.focused_pane_id, Some(right_public.clone()));
        assert_eq!(focus.layout.focused_pane_id, right_public);
        assert_eq!(app.state.workspaces[0].focused_pane_id(), Some(right));
    }

    #[test]
    fn api_pane_focus_direction_no_neighbor_is_noop() {
        let mut app = app_with_linked_worktree();
        let root = app.state.workspaces[0].tabs[0].root_pane;
        app.state.workspaces[0].tabs[0].layout.focus_pane(root);
        crate::ui::compute_view(&mut app.state, ratatui::layout::Rect::new(0, 0, 100, 20));
        let root_public = app.public_pane_id(0, root).unwrap();

        let response = app.handle_pane_focus_direction(
            "req".into(),
            crate::api::schema::PaneFocusDirectionParams {
                pane_id: Some(root_public.clone()),
                direction: PaneDirection::Left,
            },
        );

        let success: SuccessResponse = serde_json::from_str(&response).unwrap();
        let ResponseResult::PaneFocusDirection { focus } = success.result else {
            panic!("expected pane focus direction response");
        };
        assert!(!focus.changed);
        assert_eq!(focus.reason, Some(PaneFocusDirectionReason::NoNeighbor));
        assert_eq!(focus.source_pane_id, root_public.clone());
        assert_eq!(focus.focused_pane_id, Some(root_public));
        assert_eq!(app.state.workspaces[0].focused_pane_id(), Some(root));
    }
}
