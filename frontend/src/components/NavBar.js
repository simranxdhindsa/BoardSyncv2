import React from 'react';
import { ArrowLeft } from 'lucide-react';

const NavBar = ({ title, showBack, onBack, leftContent, rightContent, children }) => {
  const navHeightPx = 96; // matches py-4 plus content; adjust if needed

  return (
    <div>
      <nav
        className="glass-panel border-b border-gray-200 bg-white px-6 py-4 fixed top-0 left-0 right-0 z-50"
        style={{ borderRadius: '0', position: 'fixed', top: 0, left: 0, right: 0, zIndex: 1001 }}
      >
        <div className="flex items-center justify-between">
          {leftContent ? (
            <div className="flex items-center space-x-8">{leftContent}</div>
          ) : (
            <div className="flex items-center space-x-8">
              <div className="flex items-center">
                <img
                  src="https://apyhub.com/logo.svg"
                  alt="ApyHub"
                  className="h-8 w-8 apyhub-logo"
                />
                <span className="ml-3 text-xl font-semibold text-gray-900">
                  {title || 'BoardSync'}
                </span>
              </div>
            </div>
          )}

          <div className="flex items-center space-x-6">
            {rightContent}
            {showBack && (
              <button
                onClick={onBack}
                className="flex items-center bg-gray-100 text-gray-700 px-4 py-2 rounded-lg hover:bg-gray-200 transition-colors"
              >
                <ArrowLeft className="w-4 h-4 mr-2" />
                Back to Dashboard
              </button>
            )}
          </div>
        </div>
      </nav>

      <div
        className="max-w-6xl mx-auto px-6"
        style={{ paddingTop: navHeightPx, paddingBottom: 32 }}
      >
        {children}
      </div>
    </div>
  );
};

export default NavBar;


