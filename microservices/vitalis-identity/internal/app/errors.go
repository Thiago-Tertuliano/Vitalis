package app

import "errors"

var (
	ErrCredenciaisInvalidas = errors.New("credenciais inválidas")
	ErrUsuarioInativo       = errors.New("usuário inativo")
	ErrEmailJaExiste        = errors.New("email já existe")
	ErrRoleInvalida         = errors.New("role inválida")
	ErrRefreshInvalido      = errors.New("refresh inválido ou revogado")
	ErrNaoAutenticado       = errors.New("não autenticado")
	ErrForbidden            = errors.New("sem permissão")
	ErrResetInvalido        = errors.New("token de reset inválido")
	ErrNotFound             = errors.New("não encontrado")
)
