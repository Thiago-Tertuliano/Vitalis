package domain

type Address struct {
	ID 			string
	UserID		string
	Label		string
	CEP			string
	Logradouro	string
	Numero		string
	Complemento	string
	Bairro		string
	Cidade		string
	UF	 		string
	IsDefault 	bool
}