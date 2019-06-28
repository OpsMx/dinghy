// Code generated by MockGen. DO NOT EDIT.
// Source: builder.go

// Package dinghyfile is a generated GoMock package.
package dinghyfile

import (
	bytes "bytes"
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockParser is a mock of Renderer interface
type MockParser struct {
	ctrl     *gomock.Controller
	recorder *MockRendererMockRecorder
}

// MockRendererMockRecorder is the mock recorder for MockParser
type MockRendererMockRecorder struct {
	mock *MockParser
}

// NewMockRenderer creates a new mock instance
func NewMockRenderer(ctrl *gomock.Controller) *MockParser {
	mock := &MockParser{ctrl: ctrl}
	mock.recorder = &MockRendererMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockParser) EXPECT() *MockRendererMockRecorder {
	return m.recorder
}

// Render mocks base method
func (m *MockParser) Parse(org, repo, path string, vars []varMap) (*bytes.Buffer, error) {
	ret := m.ctrl.Call(m, "Parse", org, repo, path, vars)
	ret0, _ := ret[0].(*bytes.Buffer)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Render indicates an expected call of Render
func (mr *MockRendererMockRecorder) Parse(org, repo, path, vars interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Parse", reflect.TypeOf((*MockParser)(nil).Parse), org, repo, path, vars)
}

// MockDependencyManager is a mock of DependencyManager interface
type MockDependencyManager struct {
	ctrl     *gomock.Controller
	recorder *MockDependencyManagerMockRecorder
}

// MockDependencyManagerMockRecorder is the mock recorder for MockDependencyManager
type MockDependencyManagerMockRecorder struct {
	mock *MockDependencyManager
}

// NewMockDependencyManager creates a new mock instance
func NewMockDependencyManager(ctrl *gomock.Controller) *MockDependencyManager {
	mock := &MockDependencyManager{ctrl: ctrl}
	mock.recorder = &MockDependencyManagerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockDependencyManager) EXPECT() *MockDependencyManagerMockRecorder {
	return m.recorder
}

// SetDeps mocks base method
func (m *MockDependencyManager) SetDeps(parent string, deps []string) {
	m.ctrl.Call(m, "SetDeps", parent, deps)
}

// SetDeps indicates an expected call of SetDeps
func (mr *MockDependencyManagerMockRecorder) SetDeps(parent, deps interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetDeps", reflect.TypeOf((*MockDependencyManager)(nil).SetDeps), parent, deps)
}

// GetRoots mocks base method
func (m *MockDependencyManager) GetRoots(child string) []string {
	ret := m.ctrl.Call(m, "GetRoots", child)
	ret0, _ := ret[0].([]string)
	return ret0
}

// GetRoots indicates an expected call of GetRoots
func (mr *MockDependencyManagerMockRecorder) GetRoots(child interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRoots", reflect.TypeOf((*MockDependencyManager)(nil).GetRoots), child)
}

// MockDownloader is a mock of Downloader interface
type MockDownloader struct {
	ctrl     *gomock.Controller
	recorder *MockDownloaderMockRecorder
}

// MockDownloaderMockRecorder is the mock recorder for MockDownloader
type MockDownloaderMockRecorder struct {
	mock *MockDownloader
}

// NewMockDownloader creates a new mock instance
func NewMockDownloader(ctrl *gomock.Controller) *MockDownloader {
	mock := &MockDownloader{ctrl: ctrl}
	mock.recorder = &MockDownloaderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockDownloader) EXPECT() *MockDownloaderMockRecorder {
	return m.recorder
}

// Download mocks base method
func (m *MockDownloader) Download(org, repo, file string) (string, error) {
	ret := m.ctrl.Call(m, "Download", org, repo, file)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Download indicates an expected call of Download
func (mr *MockDownloaderMockRecorder) Download(org, repo, file interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Download", reflect.TypeOf((*MockDownloader)(nil).Download), org, repo, file)
}

// EncodeURL mocks base method
func (m *MockDownloader) EncodeURL(org, repo, file string) string {
	ret := m.ctrl.Call(m, "EncodeURL", org, repo, file)
	ret0, _ := ret[0].(string)
	return ret0
}

// EncodeURL indicates an expected call of EncodeURL
func (mr *MockDownloaderMockRecorder) EncodeURL(org, repo, file interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EncodeURL", reflect.TypeOf((*MockDownloader)(nil).EncodeURL), org, repo, file)
}

// DecodeURL mocks base method
func (m *MockDownloader) DecodeURL(url string) (string, string, string) {
	ret := m.ctrl.Call(m, "DecodeURL", url)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(string)
	ret2, _ := ret[2].(string)
	return ret0, ret1, ret2
}

// DecodeURL indicates an expected call of DecodeURL
func (mr *MockDownloaderMockRecorder) DecodeURL(url interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DecodeURL", reflect.TypeOf((*MockDownloader)(nil).DecodeURL), url)
}
