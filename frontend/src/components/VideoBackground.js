import React, { useEffect, useRef, useState } from 'react';

const VideoBackground = ({ 
  videoSrc = '/assets/background-video.mp4',
  fallbackImage = '/assets/fallback-background.jpg',
  opacity = 1,
  blur = 1,
  enabled = true 
}) => {
  const videoRef = useRef(null);
  const [videoLoaded, setVideoLoaded] = useState(false);
  const [videoError, setVideoError] = useState(false);

  useEffect(() => {
    const video = videoRef.current;
    if (!video || !enabled) return;

    const handleLoadedData = () => {
      setVideoLoaded(true);
      setVideoError(false);
    };

    const handleError = () => {
      setVideoError(true);
      setVideoLoaded(false);
    };

    const handleCanPlay = () => {
      video.play().catch(() => setVideoError(true));
    };

    video.addEventListener('loadeddata', handleLoadedData);
    video.addEventListener('error', handleError);
    video.addEventListener('canplay', handleCanPlay);

    return () => {
      video.removeEventListener('loadeddata', handleLoadedData);
      video.removeEventListener('error', handleError);
      video.removeEventListener('canplay', handleCanPlay);
    };
  }, [enabled]);

  if (!enabled) return null;

  return (
    <>
      {/* Video Container */}
      <div 
        style={{
          position: 'fixed',
          top: 0,
          left: 0,
          width: '100vw',
          height: '100vh',
          zIndex: -10, // Very low z-index to stay behind everything
          overflow: 'hidden',
          pointerEvents: 'none'
        }}
      >
        {/* Video Element */}
        {videoSrc && !videoError && (
          <video
            ref={videoRef}
            style={{
              position: 'absolute',
              top: '50%',
              left: '50%',
              minWidth: '100%',
              minHeight: '100%',
              width: 'auto',
              height: 'auto',
              transform: 'translate(-50%, -50%)',
              opacity: videoLoaded ? opacity : 0,
              filter: `blur(${blur}px)`,
              transition: 'opacity 1s ease-in-out',
              objectFit: 'cover'
            }}
            autoPlay
            muted
            loop
            playsInline
            preload="auto"
          >
            <source src={videoSrc} type="video/mp4" />
          </video>
        )}

        {/* Fallback Image */}
        {(videoError || !videoSrc) && fallbackImage && (
          <div
            style={{
              position: 'absolute',
              top: 0,
              left: 0,
              width: '100%',
              height: '100%',
              backgroundImage: `url(${fallbackImage})`,
              backgroundSize: 'cover',
              backgroundPosition: 'center',
              backgroundRepeat: 'no-repeat',
              filter: `blur(${blur}px)`,
              opacity: opacity
            }}
          />
        )}
      </div>      
    </>
  );
};

export default VideoBackground;