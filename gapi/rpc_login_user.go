package gapi

import (
	"context"
	"errors"

	db "github.com/a7medalyapany/GoBank.git/db/sqlc"
	"github.com/a7medalyapany/GoBank.git/pb"
	"github.com/a7medalyapany/GoBank.git/util"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)


func (server *Server) LoginUser(ctx context.Context, req *pb.LoginUserRequest) (*pb.LoginUserResponse, error) {

	user, err := server.store.GetUser(ctx, req.GetUsername())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
		return nil, status.Errorf(codes.NotFound, "User not found: %v", err)
		}
		return nil, status.Errorf(codes.Internal, "Failed to get user: %v", err)
	}

	err = util.CheckPassword(req.Password, user.HashedPassword)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "Invalid password: %v", err)
	}

	accessToken, accessPayload, err := server.tokenMaker.CreateToken(
		user.Username,
		server.config.ACCESS_TOKEN_DURATION,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to create access token: %v", err)
	}

	refreshToken, refreshPayload, err := server.tokenMaker.CreateToken(
		user.Username,
		server.config.REFRESH_TOKEN_DURATION,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to create refresh token: %v", err)
	}

	mt := server.extractMetadata(ctx)

	session, err := server.store.CreateSession(ctx, db.CreateSessionParams{
		ID: pgtype.UUID{Bytes: refreshPayload.ID, Valid: true},
		Username: user.Username,
		RefreshToken: refreshToken,
		UserAgent: mt.UserAgent,
		ClientIp: mt.ClientIp,
		IsBlocked: false,
		ExpiresAt: pgtype.Timestamptz{Time: refreshPayload.ExpiredAt, Valid: true},
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to create session: %v", err)
	}


	rsp := &pb.LoginUserResponse{
		SessionId: session.ID.String(),
		AccessToken: accessToken,
		RefreshToken: refreshToken,
		AccessTokenExpiresAt: timestamppb.New(accessPayload.ExpiredAt),
		RefreshTokenExpiresAt: timestamppb.New(refreshPayload.ExpiredAt),

		User: convertUser(user),
	}
	return rsp, nil
}