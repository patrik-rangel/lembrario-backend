package worker                                                                                                                                               
                                                                                                                                                             
import (                                                                                                                                                     
    "context"                                                                                                                                                
    "fmt"                                                                                                                                                    
    "net/http"                                                                                                                                               
    "net/url"                                                                                                                                                
    "strings"                                                                                                                                                
    "time"                                                                                                                                                   
                                                                                                                                                             
    "github.com/PuerkitoBio/goquery"                                                                                                                         
)                                                                                                                                                            
                                                                                                                                                             
// ScrapedData representa os dados extraídos de uma URL                                                                                                      
type ScrapedData struct {                                                                                                                                    
    Title       string                                                                                                                                       
    Description string                                                                                                                                       
    Provider    string                                                                                                                                       
}                                                                                                                                                            
                                                                                                                                                             
// ScrapeURL extrai metadados básicos de uma URL usando goquery                                                                                              
func ScrapeURL(ctx context.Context, targetURL string) (*ScrapedData, error) {                                                                                
    // Validar URL                                                                                                                                           
    parsedURL, err := url.Parse(targetURL)                                                                                                                   
    if err != nil {                                                                                                                                          
        return nil, fmt.Errorf("URL inválida: %w", err)                                                                                                      
    }                                                                                                                                                        
                                                                                                                                                             
    // Criar cliente HTTP com timeout                                                                                                                        
    client := &http.Client{                                                                                                                                  
        Timeout: 30 * time.Second,                                                                                                                           
    }                                                                                                                                                        
                                                                                                                                                             
    // Criar request com contexto                                                                                                                            
    req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)                                                                                       
    if err != nil {                                                                                                                                          
        return nil, fmt.Errorf("erro ao criar request: %w", err)                                                                                             
    }                                                                                                                                                        
                                                                                                                                                             
    // Adicionar User-Agent para evitar bloqueios                                                                                                            
    req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Lembrario/1.0)")                                                                                  
                                                                                                                                                             
    // Fazer a requisição                                                                                                                                    
    resp, err := client.Do(req)                                                                                                                              
    if err != nil {                                                                                                                                          
        return nil, fmt.Errorf("erro ao fazer requisição HTTP: %w", err)                                                                                     
    }                                                                                                                                                        
    defer resp.Body.Close()                                                                                                                                  
                                                                                                                                                             
    // Verificar status code                                                                                                                                 
    if resp.StatusCode != http.StatusOK {                                                                                                                    
        return nil, fmt.Errorf("status HTTP não OK: %d", resp.StatusCode)                                                                                    
    }                                                                                                                                                        
                                                                                                                                                             
    // Parse do HTML com goquery                                                                                                                             
    doc, err := goquery.NewDocumentFromReader(resp.Body)                                                                                                     
    if err != nil {                                                                                                                                          
        return nil, fmt.Errorf("erro ao fazer parse do HTML: %w", err)                                                                                       
    }                                                                                                                                                        
                                                                                                                                                             
    // Extrair metadados                                                                                                                                     
    data := &ScrapedData{                                                                                                                                    
        Provider: identifyProvider(parsedURL.Host),                                                                                                          
    }                                                                                                                                                        
                                                                                                                                                             
    // Extrair título                                                                                                                                        
    data.Title = strings.TrimSpace(doc.Find("title").First().Text())                                                                                         
                                                                                                                                                             
    // Extrair descrição (meta description)                                                                                                                  
    description, exists := doc.Find("meta[name='description']").First().Attr("content")                                                                      
    if exists {                                                                                                                                              
        data.Description = strings.TrimSpace(description)                                                                                                    
    }                                                                                                                                                        
                                                                                                                                                             
    // Se não encontrou description, tentar og:description                                                                                                   
    if data.Description == "" {                                                                                                                              
        ogDescription, exists := doc.Find("meta[property='og:description']").First().Attr("content")                                                         
        if exists {                                                                                                                                          
            data.Description = strings.TrimSpace(ogDescription)                                                                                              
        }                                                                                                                                                    
    }                                                                                                                                                        
                                                                                                                                                             
    return data, nil                                                                                                                                         
}                                                                                                                                                            
                                                                                                                                                             
// identifyProvider identifica o provedor baseado no hostname                                                                                                
func identifyProvider(host string) string {                                                                                                                  
    host = strings.ToLower(host)                                                                                                                             
                                                                                                                                                             
    switch {                                                                                                                                                 
    case strings.Contains(host, "youtube.com") || strings.Contains(host, "youtu.be"):                                                                        
        return "youtube"                                                                                                                                     
    case strings.Contains(host, "vimeo.com"):                                                                                                                
        return "vimeo"                                                                                                                                       
    case strings.Contains(host, "medium.com"):                                                                                                               
        return "medium"                                                                                                                                      
    case strings.Contains(host, "github.com"):                                                                                                               
        return "github"                                                                                                                                      
    case strings.Contains(host, "twitter.com") || strings.Contains(host, "x.com"):                                                                           
        return "twitter"                                                                                                                                     
    case strings.Contains(host, "linkedin.com"):                                                                                                             
        return "linkedin"                                                                                                                                    
    default:                                                                                                                                                 
        return "web"                                                                                                                                         
    }                                                                                                                                                        
}          