import React, { useEffect, useRef } from 'react';
import * as THREE from 'three';

const ThreeBackground = ({ currentView, analysisData, selectedColumn, isLoading }) => {
  const mountRef = useRef(null);
  const sceneRef = useRef(null);
  const rendererRef = useRef(null);
  const cameraRef = useRef(null);
  const animationIdRef = useRef(null);
  const mouseRef = useRef({ x: 0, y: 0, normalizedX: 0, normalizedY: 0 });
  const movingElementsRef = useRef([]);
  const currentThemeRef = useRef(0);
  const themeTimerRef = useRef(0);
  const spawnTimerRef = useRef(0);
  
  // All tech themes that cycle
  const techThemes = [
    'binary_streams',
    'data_packets',
    'signal_waves', 
    'progress_bars',
    'code_snippets',
    'git_commits',
    'api_endpoints',
    'database_symbols',
    'network_nodes',
    'server_racks',
    'cloud_symbols',
    'circuit_traces'
  ];

  // Simple, clean element creators
  const createBinaryStream = () => {
    const geometry = new THREE.PlaneGeometry(1, 1.5);
    const material = new THREE.MeshBasicMaterial({
      color: Math.random() > 0.5 ? 0x00ff41 : 0x008f11,
      transparent: true,
      opacity: 0.8
    });
    return new THREE.Mesh(geometry, material);
  };

  const createDataPacket = () => {
    const geometry = new THREE.BoxGeometry(2, 1, 1);
    const material = new THREE.MeshBasicMaterial({
      color: 0x3b82f6,
      transparent: true,
      opacity: 0.7,
      wireframe: true
    });
    return new THREE.Mesh(geometry, material);
  };

  const createSignalWave = () => {
    const points = [];
    for (let i = 0; i <= 30; i++) {
      const x = (i / 30) * 6 - 3;
      const y = Math.sin(x * 2) * 1.5;
      points.push(new THREE.Vector3(x, y, 0));
    }
    const geometry = new THREE.BufferGeometry().setFromPoints(points);
    const material = new THREE.LineBasicMaterial({ 
      color: 0x06b6d4,
      linewidth: 2
    });
    return new THREE.Line(geometry, material);
  };

  const createProgressBar = () => {
    const group = new THREE.Group();
    const progress = Math.random();
    
    // Background
    const bgGeometry = new THREE.BoxGeometry(4, 0.4, 0.2);
    const bgMaterial = new THREE.MeshBasicMaterial({ 
      color: 0x333333,
      transparent: true,
      opacity: 0.5
    });
    const bg = new THREE.Mesh(bgGeometry, bgMaterial);
    group.add(bg);
    
    // Fill
    const fillGeometry = new THREE.BoxGeometry(4 * progress, 0.3, 0.15);
    const fillMaterial = new THREE.MeshBasicMaterial({ 
      color: progress > 0.7 ? 0x10b981 : progress > 0.4 ? 0xf59e0b : 0xef4444
    });
    const fill = new THREE.Mesh(fillGeometry, fillMaterial);
    fill.position.x = -(4 - 4 * progress) / 2;
    group.add(fill);
    
    return group;
  };

  const createCodeSnippet = () => {
    const group = new THREE.Group();
    for (let i = 0; i < 4; i++) {
      const width = 1 + Math.random() * 2;
      const geometry = new THREE.BoxGeometry(width, 0.3, 0.1);
      const material = new THREE.MeshBasicMaterial({ 
        color: 0xf59e0b,
        transparent: true,
        opacity: 0.8
      });
      const line = new THREE.Mesh(geometry, material);
      line.position.set(0, i * 0.4 - 0.6, 0);
      group.add(line);
    }
    return group;
  };

  const createGitCommit = () => {
    const group = new THREE.Group();
    
    // Main line
    const lineGeometry = new THREE.BoxGeometry(4, 0.1, 0.1);
    const lineMaterial = new THREE.MeshBasicMaterial({ color: 0x10b981 });
    const line = new THREE.Mesh(lineGeometry, lineMaterial);
    group.add(line);
    
    // Commit nodes
    for (let i = 0; i < 5; i++) {
      const nodeGeometry = new THREE.SphereGeometry(0.2);
      const nodeMaterial = new THREE.MeshBasicMaterial({ color: 0x10b981 });
      const node = new THREE.Mesh(nodeGeometry, nodeMaterial);
      node.position.x = i * 1 - 2;
      group.add(node);
    }
    
    return group;
  };

  const createAPIEndpoint = () => {
    const group = new THREE.Group();
    
    const boxGeometry = new THREE.BoxGeometry(3, 0.8, 0.2);
    const boxMaterial = new THREE.MeshBasicMaterial({ 
      color: 0x06b6d4,
      wireframe: true
    });
    const box = new THREE.Mesh(boxGeometry, boxMaterial);
    group.add(box);
    
    const dotGeometry = new THREE.SphereGeometry(0.1);
    const dotMaterial = new THREE.MeshBasicMaterial({ color: 0x10b981 });
    const dot = new THREE.Mesh(dotGeometry, dotMaterial);
    dot.position.x = -1.2;
    group.add(dot);
    
    return group;
  };

  const createDatabase = () => {
    const geometry = new THREE.CylinderGeometry(1, 1, 2, 12);
    const material = new THREE.MeshBasicMaterial({
      color: 0x8b5cf6,
      transparent: true,
      opacity: 0.7,
      wireframe: true
    });
    return new THREE.Mesh(geometry, material);
  };

  const createNetworkNode = () => {
    const group = new THREE.Group();
    
    // Center node
    const centerGeometry = new THREE.SphereGeometry(0.5);
    const centerMaterial = new THREE.MeshBasicMaterial({ color: 0xf59e0b });
    const center = new THREE.Mesh(centerGeometry, centerMaterial);
    group.add(center);
    
    // Connected nodes
    for (let i = 0; i < 4; i++) {
      const angle = (i / 4) * Math.PI * 2;
      const x = Math.cos(angle) * 2;
      const y = Math.sin(angle) * 2;
      
      const nodeGeometry = new THREE.SphereGeometry(0.2);
      const nodeMaterial = new THREE.MeshBasicMaterial({ color: 0x06b6d4 });
      const node = new THREE.Mesh(nodeGeometry, nodeMaterial);
      node.position.set(x, y, 0);
      group.add(node);
      
      // Connection line
      const lineGeometry = new THREE.BufferGeometry().setFromPoints([
        new THREE.Vector3(0, 0, 0),
        new THREE.Vector3(x, y, 0)
      ]);
      const lineMaterial = new THREE.LineBasicMaterial({ 
        color: 0xf59e0b,
        transparent: true,
        opacity: 0.6
      });
      const line = new THREE.Line(lineGeometry, lineMaterial);
      group.add(line);
    }
    
    return group;
  };

  const createServerRack = () => {
    const group = new THREE.Group();
    
    // Main frame
    const frameGeometry = new THREE.BoxGeometry(2, 4, 1);
    const frameMaterial = new THREE.MeshBasicMaterial({ 
      color: 0x6366f1,
      wireframe: true
    });
    const frame = new THREE.Mesh(frameGeometry, frameMaterial);
    group.add(frame);
    
    // Servers
    for (let i = 0; i < 6; i++) {
      const serverGeometry = new THREE.BoxGeometry(1.8, 0.5, 0.8);
      const serverMaterial = new THREE.MeshBasicMaterial({ 
        color: i % 2 === 0 ? 0x10b981 : 0xf59e0b 
      });
      const server = new THREE.Mesh(serverGeometry, serverMaterial);
      server.position.y = i * 0.6 - 1.5;
      group.add(server);
    }
    
    return group;
  };

  const createCloudSymbol = () => {
    const group = new THREE.Group();
    
    // Cloud spheres
    const positions = [
      [0, 0, 0], [-1, 0, 0], [1, 0, 0], [-0.5, 0.8, 0], [0.5, 0.8, 0]
    ];
    
    positions.forEach(pos => {
      const geometry = new THREE.SphereGeometry(0.5);
      const material = new THREE.MeshBasicMaterial({ 
        color: 0x06b6d4,
        wireframe: true,
        transparent: true,
        opacity: 0.7
      });
      const sphere = new THREE.Mesh(geometry, material);
      sphere.position.set(...pos);
      group.add(sphere);
    });
    
    return group;
  };

  const createCircuitTrace = () => {
    const group = new THREE.Group();
    
    const points = [
      new THREE.Vector3(-2, 0, 0),
      new THREE.Vector3(-1, 0, 0),
      new THREE.Vector3(-1, 1, 0),
      new THREE.Vector3(1, 1, 0),
      new THREE.Vector3(1, -1, 0),
      new THREE.Vector3(2, -1, 0)
    ];
    
    const geometry = new THREE.BufferGeometry().setFromPoints(points);
    const material = new THREE.LineBasicMaterial({ 
      color: 0x10b981,
      linewidth: 2
    });
    const trace = new THREE.Line(geometry, material);
    group.add(trace);
    
    // Connection points
    points.forEach(point => {
      const dotGeometry = new THREE.SphereGeometry(0.1);
      const dotMaterial = new THREE.MeshBasicMaterial({ color: 0x10b981 });
      const dot = new THREE.Mesh(dotGeometry, dotMaterial);
      dot.position.copy(point);
      group.add(dot);
    });
    
    return group;
  };

  // Get current theme creator
  const getCurrentThemeCreator = () => {
    const currentTheme = techThemes[currentThemeRef.current];
    
    switch(currentTheme) {
      case 'binary_streams': return createBinaryStream;
      case 'data_packets': return createDataPacket;
      case 'signal_waves': return createSignalWave;
      case 'progress_bars': return createProgressBar;
      case 'code_snippets': return createCodeSnippet;
      case 'git_commits': return createGitCommit;
      case 'api_endpoints': return createAPIEndpoint;
      case 'database_symbols': return createDatabase;
      case 'network_nodes': return createNetworkNode;
      case 'server_racks': return createServerRack;
      case 'cloud_symbols': return createCloudSymbol;
      case 'circuit_traces': return createCircuitTrace;
      default: return createDataPacket;
    }
  };

  // Clear all elements
  const clearAllElements = () => {
    movingElementsRef.current.forEach(element => {
      sceneRef.current.remove(element.object);
      if (element.object.traverse) {
        element.object.traverse(child => {
          if (child.geometry) child.geometry.dispose();
          if (child.material) {
            if (Array.isArray(child.material)) {
              child.material.forEach(mat => mat.dispose());
            } else {
              child.material.dispose();
            }
          }
        });
      }
    });
    movingElementsRef.current = [];
  };

  // Spawn individual element with proper spacing
  const spawnElement = () => {
    const creator = getCurrentThemeCreator();
    const element = creator();
    
    // Start position - far left, random Y with good spacing
    element.position.set(
      -100, // Start far left
      (Math.random() - 0.5) * 50, // Random Y position with wide spread
      (Math.random() - 0.5) * 20  // Random Z for depth
    );
    
    // Random but subtle rotation
    element.rotation.set(
      (Math.random() - 0.5) * 0.5,
      (Math.random() - 0.5) * 0.5,
      (Math.random() - 0.5) * 0.5
    );
    
    // Scale for visibility
    const scale = 2 + Math.random() * 2;
    element.scale.setScalar(scale);
    
    sceneRef.current.add(element);
    
    // Store with movement properties
    movingElementsRef.current.push({
      object: element,
      speed: 0.3 + Math.random() * 0.4,
      rotationSpeed: (Math.random() - 0.5) * 0.01,
      verticalFloat: Math.random() * 0.15,
      floatOffset: Math.random() * Math.PI * 2,
      mouseResponseX: (Math.random() - 0.5) * 2,
      mouseResponseY: (Math.random() - 0.5) * 2
    });
  };

  // Create steady background elements
  const createStaticElements = () => {
    clearAllElements();
    const creator = getCurrentThemeCreator();
    
    // Create multiple elements spread across the screen
    for (let i = 0; i < 12; i++) {
      const element = creator();
      
      // Distribute elements across the entire screen with good spacing
      element.position.set(
        (Math.random() - 0.5) * 120, // Spread across X (-60 to +60)
        (Math.random() - 0.5) * 60,  // Spread across Y (-30 to +30)
        (Math.random() - 0.5) * 40   // Random Z depth (-20 to +20)
      );
      
      // Random but subtle rotation
      element.rotation.set(
        (Math.random() - 0.5) * 0.5,
        (Math.random() - 0.5) * 0.5,
        (Math.random() - 0.5) * 0.5
      );
      
      // Scale for visibility
      const scale = 1.5 + Math.random() * 1.5;
      element.scale.setScalar(scale);
      
      sceneRef.current.add(element);
      
      // Store with animation properties (no horizontal movement)
      movingElementsRef.current.push({
        object: element,
        rotationSpeed: (Math.random() - 0.5) * 0.008,
        verticalFloat: Math.random() * 0.1,
        floatOffset: Math.random() * Math.PI * 2,
        mouseResponseX: (Math.random() - 0.5) * 1,
        mouseResponseY: (Math.random() - 0.5) * 1,
        originalY: element.position.y
      });
    }
  };

  // Create all elements at once with high scaling
  const createAllElements = () => {
    clearAllElements();
    
    // Array of all element creators
    const allCreators = [
      createBinaryStream,
      createDataPacket,
      createSignalWave, 
      createProgressBar,
      createCodeSnippet,
      createGitCommit,
      createAPIEndpoint,
      createDatabase,
      createNetworkNode,
      createServerRack,
      createCloudSymbol,
      createCircuitTrace
    ];
    
    // Create multiple instances of each element type
    allCreators.forEach((creator, typeIndex) => {
      for (let i = 0; i < 8; i++) { // 8 instances of each type
        const element = creator();
        
        // Distribute elements across the entire screen with good spacing
        element.position.set(
          (Math.random() - 0.5) * 160, // Wide spread across X (-80 to +80)
          (Math.random() - 0.5) * 80,  // Wide spread across Y (-40 to +40)
          (Math.random() - 0.5) * 60   // Deep Z spread (-30 to +30)
        );
        
        // Random rotation for variety
        element.rotation.set(
          (Math.random() - 0.5) * 1,
          (Math.random() - 0.5) * 1,
          (Math.random() - 0.5) * 1
        );
        
        // High scaling for visibility
        const scale = 3 + Math.random() * 3; // Scale 3-6x for high visibility
        element.scale.setScalar(scale);
        
        sceneRef.current.add(element);
        
        // Store with animation properties
        movingElementsRef.current.push({
          object: element,
          rotationSpeed: (Math.random() - 0.5) * 0.005,
          verticalFloat: Math.random() * 0.08,
          floatOffset: Math.random() * Math.PI * 2,
          mouseResponseX: (Math.random() - 0.5) * 0.8,
          mouseResponseY: (Math.random() - 0.5) * 0.8,
          originalY: element.position.y,
          elementType: typeIndex
        });
      }
    });
    
    console.log(`Created ${movingElementsRef.current.length} total elements (all types visible)`);
  };

  useEffect(() => {
    console.log('ThreeBackground: Starting improved tech system...');
    initScene();
    startAnimation();

    const handleMouseMove = (event) => {
      mouseRef.current = {
        normalizedX: (event.clientX / window.innerWidth) * 2 - 1,
        normalizedY: -(event.clientY / window.innerHeight) * 2 + 1
      };
    };

    const handleResize = () => {
      if (cameraRef.current && rendererRef.current) {
        cameraRef.current.aspect = window.innerWidth / window.innerHeight;
        cameraRef.current.updateProjectionMatrix();
        rendererRef.current.setSize(window.innerWidth, window.innerHeight);
      }
    };

    window.addEventListener('mousemove', handleMouseMove);
    window.addEventListener('resize', handleResize);

    return () => {
      window.removeEventListener('mousemove', handleMouseMove);
      window.removeEventListener('resize', handleResize);
      cleanup();
    };
  }, []);

  const initScene = () => {
    const scene = new THREE.Scene();
    scene.background = null;
    sceneRef.current = scene;

    // Camera positioned for good left-to-right view
    const camera = new THREE.PerspectiveCamera(75, window.innerWidth / window.innerHeight, 0.1, 2000);
    camera.position.set(0, 0, 60);
    camera.lookAt(0, 0, 0);
    cameraRef.current = camera;

    const renderer = new THREE.WebGLRenderer({ alpha: true, antialias: true });
    renderer.setSize(window.innerWidth, window.innerHeight);
    renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
    renderer.setClearColor(0x000000, 0);
    rendererRef.current = renderer;

    if (mountRef.current) {
      mountRef.current.appendChild(renderer.domElement);
    }

    // Ambient lighting for visibility
    const ambientLight = new THREE.AmbientLight(0x404040, 0.8);
    scene.add(ambientLight);

    const directionalLight = new THREE.DirectionalLight(0xffffff, 1.0);
    directionalLight.position.set(10, 10, 10);
    scene.add(directionalLight);
  };

  const startAnimation = () => {
    animate();
  };

  const animate = () => {
    animationIdRef.current = requestAnimationFrame(animate);

    const time = Date.now() * 0.001;
    
    // Create all elements on first load or if none exist
    if (movingElementsRef.current.length === 0) {
      createAllElements();
    }
    
    // Animate all elements (no theme switching - show all at once)
    const mouse = mouseRef.current;
    movingElementsRef.current.forEach(element => {
      // Gentle floating motion
      element.object.position.y = element.originalY + Math.sin(time * 1.2 + element.floatOffset) * element.verticalFloat;
      
      // Very subtle rotation
      element.object.rotation.y += element.rotationSpeed;
      element.object.rotation.z += element.rotationSpeed * 0.3;
      
      // Mouse interaction - subtle response to cursor
      const mouseInfluenceX = mouse.normalizedX * element.mouseResponseX * 0.3;
      const mouseInfluenceY = mouse.normalizedY * element.mouseResponseY * 0.2;
      
      element.object.position.y += mouseInfluenceY;
      element.object.rotation.y += mouseInfluenceX * 0.003;
    });
    
    // Mouse-responsive camera with wider range for high scaling
    if (cameraRef.current) {
      const targetX = mouse.normalizedX * 4;
      const targetY = -mouse.normalizedY * 3;
      
      cameraRef.current.position.x += (targetX - cameraRef.current.position.x) * 0.02;
      cameraRef.current.position.y += (targetY - cameraRef.current.position.y) * 0.02;
      cameraRef.current.lookAt(0, 0, 0);
    }

    if (rendererRef.current && sceneRef.current && cameraRef.current) {
      rendererRef.current.render(sceneRef.current, cameraRef.current);
    }
  };

  const cleanup = () => {
    if (animationIdRef.current) {
      cancelAnimationFrame(animationIdRef.current);
    }
    
    clearAllElements();
    
    if (rendererRef.current && mountRef.current && mountRef.current.contains(rendererRef.current.domElement)) {
      mountRef.current.removeChild(rendererRef.current.domElement);
      rendererRef.current.dispose();
    }
  };

  return (
    <div 
      ref={mountRef}
      style={{
        position: 'fixed',
        top: 0,
        left: 0,
        width: '100vw',
        height: '100vh',
        zIndex: -1,
        pointerEvents: 'none',
        backgroundColor: 'transparent'
      }}
    />
  );
};

export default ThreeBackground;