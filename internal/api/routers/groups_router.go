package routers

import (
	"net/http"
	"qiyana_paybuddy/internal/api/handlers/groups"
)

func groupsRouter() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/groups/create", groups.CreateGroupHandler)

	mux.HandleFunc("/groups/", groups.GetMyGroupsHandler)

	mux.HandleFunc("/groups/{id}", groups.GetGroupByIDHandler)

	mux.HandleFunc("/groups/delete/{id}", groups.DeleteGroupByHandler)

	mux.HandleFunc("/groups/update/{id}", groups.UpdateGroupHandler)

	mux.HandleFunc("/groups/member/{id}/invite", groups.InviteMembersHandler)

	mux.HandleFunc("/groups/member/accept/{tokenCode}/invite", groups.AcceptInvitationHandler)

	mux.HandleFunc("/groups/member/{id}/remove", groups.RemoveGroupMemberHandler)

	mux.HandleFunc("/groups/member/{id}/leave", groups.LeaveGroupHandler)

	mux.HandleFunc("/groups/invites/{id}/revoke", groups.RevokeInvitationHandler)

	mux.HandleFunc("/groups/{id}/invites/pending", groups.ListPendingInvitesHandler)

	mux.HandleFunc("/groups/{groupId}/invites/{inviteId}/pending", groups.GetOnePendingInviteHandler)

	mux.HandleFunc("/groups/{groupId}/invites/{inviteId}/resend", groups.ResendInviteHandler)

	return mux
}
