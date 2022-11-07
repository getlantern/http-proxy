/*
	MIT License

	Copyright (c) 2020 Operator Foundation

	Permission is hereby granted, free of charge, to any person obtaining a copy
	of this software and associated documentation files (the "Software"), to deal
	in the Software without restriction, including without limitation the rights
	to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
	copies of the Software, and to permit persons to whom the Software is
	furnished to do so, subject to the following conditions:

	The above copyright notice and this permission notice shall be included in all
	copies or substantial portions of the Software.

	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
	IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
	FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
	AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
	LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
	OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
	SOFTWARE.
*/

package replicant

import (
	"encoding/json"
	"errors"

	"github.com/OperatorFoundation/Replicant-go/Replicant/v3/polish"
	"github.com/OperatorFoundation/Replicant-go/Replicant/v3/toneburst"
)

type ClientConfig struct {
	ServerAddress string    		  `json:"serverAddress"`
	Toneburst     toneburst.Config    `json:"toneburst"`    
	Polish        polish.ClientConfig `json:"polish"`       
	Transport     string    		  `json:"transport"`
}

type ServerConfig struct {
	ServerAddress string    		  `json:"serverAddress"`
	Toneburst     toneburst.Config    `json:"toneburst"`    
	Polish        polish.ServerConfig `json:"polish"`       
	Transport     string    		  `json:"transport"`
}

func (config ServerConfig) ToJsonString() (string, error) {
	jsonBytes, marshalError := json.MarshalIndent(config, "", "  ")
	if marshalError != nil {
		return "", marshalError
	}

	return string(jsonBytes), nil
}

func (config ClientConfig) ToJsonString() (string, error) {
	jsonBytes, marshalError := json.MarshalIndent(config, "", "  ")
	if marshalError != nil {
		return "", marshalError
	}

	return string(jsonBytes), nil
}

func (config ServerConfig) Marshal() (string, error) {
	polishConfig, ok := config.Polish.(polish.DarkStarPolishServerConfig)
	if !ok {
		return "", errors.New("polish config was not a DarkStar polish config")
	}

	toneburstConfig, ok := config.Toneburst.(toneburst.StarburstConfig)
	if !ok {
		return "", errors.New("toneburst config was not a Starburst config")
	}

	jsonConfig := ServerJsonConfig {
		ServerAddress: config.ServerAddress,
		Toneburst: toneburstConfig,
		Polish: DarkStarPolishServerJsonConfig{ServerPrivateKey: polishConfig.ServerPrivateKey},
		Transport: config.Transport,
	}

	configBytes, configStringError := json.Marshal(jsonConfig)
	if configStringError != nil {
		return "", configStringError
	}

	return string(configBytes), nil
}

func (config ClientConfig) Marshal() (string, error) {
	polishConfig, ok := config.Polish.(polish.DarkStarPolishClientConfig)
	if !ok {
		return "", errors.New("polish config was not a DarkStar polish config")
	}

	toneburstConfig, ok := config.Toneburst.(toneburst.StarburstConfig)
	if !ok {
		return "", errors.New("toneburst config was not a Starburst config")
	}

	jsonConfig := ClientJsonConfig {
		ServerAddress: config.ServerAddress,
		Toneburst: toneburstConfig,
		Polish: DarkStarPolishClientJsonConfig{ServerPublicKey: polishConfig.ServerPublicKey},
		Transport: config.Transport,
	}

	configBytes, configStringError := json.Marshal(jsonConfig)
	if configStringError != nil {
		return "", configStringError
	}

	return string(configBytes), nil
}

func UnmarshalClientConfig(data []byte) (*ClientConfig, error) {
	var clientJsonConfig ClientJsonConfig
	unmarshalError := json.Unmarshal(data, &clientJsonConfig)
	if unmarshalError != nil {
		return nil, unmarshalError
	}

	polishConfig := polish.DarkStarPolishClientConfig{
		ServerAddress: clientJsonConfig.ServerAddress,
		ServerPublicKey: clientJsonConfig.Polish.ServerPublicKey,
	}

	clientConfig := ClientConfig {
		ServerAddress: clientJsonConfig.ServerAddress,
		Toneburst: clientJsonConfig.Toneburst,
		Polish: polishConfig,
		Transport: clientJsonConfig.Transport,
	}

	return &clientConfig, nil
}

func UnmarshalServerConfig(data []byte) (*ServerConfig, error) {
	var serverJsonConfig ServerJsonConfig
	unmarshalError := json.Unmarshal(data, &serverJsonConfig)
	if unmarshalError != nil {
		return nil, unmarshalError
	}

	polishConfig := polish.DarkStarPolishServerConfig{
		ServerAddress: serverJsonConfig.ServerAddress,
		ServerPrivateKey: serverJsonConfig.Polish.ServerPrivateKey,
	}

	serverConfig := ServerConfig {
		ServerAddress: serverJsonConfig.ServerAddress,
		Toneburst: serverJsonConfig.Toneburst,
		Polish: polishConfig,
		Transport: serverJsonConfig.Transport,
	}

	return &serverConfig, nil
}